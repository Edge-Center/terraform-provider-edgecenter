//go:build integration

package edgecenter_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func scopeListedProjects() []edgecloud.Project {
	return []edgecloud.Project{
		{ID: testProjectID, Name: "test-project"},
		{ID: testOtherProjectID, Name: "other-project"},
	}
}

func scopeCloudFailingPlanOnly() *cloudmock.MockedCloud {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return(nil, nil, errors.New("list failed")).Once()
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return(scopeListedProjects(), nil, nil)
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{{ID: testRegionID, DisplayName: "test-region"}}, nil, nil).Maybe()

	return mc
}

func TestIntegrationCloudScopeNameOnUpdate(t *testing.T) {
	t.Parallel()

	resources := map[string]map[string]interface{}{
		"edgecenter_volume": cloud.Merge(
			cloud.WithName("test-volume"),
			cloud.WithSize(1),
			cloud.WithTypeName("standard"),
		),
		"edgecenter_snapshot": cloud.Merge(
			cloud.WithName("test-snapshot"),
			map[string]interface{}{"volume_id": "vol-1"},
		),
		"edgecenter_instanceV2": cloud.Merge(
			cloud.WithName("test-instance"),
			map[string]interface{}{"flavor_id": "g1-standard-2-4"},
		),
		"edgecenter_instance": cloud.Merge(
			cloud.WithName("test-instance"),
			map[string]interface{}{"flavor_id": "g1-standard-2-4"},
		),
		"edgecenter_baremetal": cloud.Merge(
			cloud.WithName("test-baremetal"),
			map[string]interface{}{"flavor_id": "bm1-infrastructure-small"},
		),
		"edgecenter_network": cloud.Merge(
			cloud.WithName("test-network"),
			map[string]interface{}{"type": "vxlan"},
		),
		"edgecenter_subnet": cloud.Merge(
			cloud.WithName("test-subnet"),
			map[string]interface{}{"cidr": "192.168.10.0/24", "network_id": "net-1"},
		),
		"edgecenter_securitygroup":   cloud.WithName("test-sg"),
		"edgecenter_router":          cloud.WithName("test-router"),
		"edgecenter_floatingip":      {},
		"edgecenter_reservedfixedip": {"type": "external"},
	}

	names := map[string]interface{}{
		"project_name": "test-project",
		"region_name":  "test-region",
	}
	unchanged := map[string]string{
		"project_id":   fmt.Sprint(testProjectID),
		"project_name": "test-project",
		"region_id":    fmt.Sprint(testRegionID),
		"region_name":  "test-region",
	}

	cases := []struct {
		name    string
		config  map[string]interface{}
		prepare func() *cloudmock.MockedCloud
		wantErr string
	}{
		{
			name:    "project name that does not resolve",
			config:  map[string]interface{}{"project_name": "typo-project", "region_name": "test-region"},
			prepare: func() *cloudmock.MockedCloud { return scopeResolvingCloud(nil) },
			wantErr: `cannot resolve project_name "typo-project"`,
		},
		{
			name:    "region name that does not resolve",
			config:  map[string]interface{}{"project_name": "test-project", "region_name": "typo-region"},
			prepare: func() *cloudmock.MockedCloud { return scopeResolvingCloud(nil) },
			wantErr: `cannot resolve region_name "typo-region"`,
		},
		{
			name:    "project name while the lists are unavailable",
			config:  map[string]interface{}{"project_name": "other-project", "region_name": "test-region"},
			prepare: func() *cloudmock.MockedCloud { return scopeResolvingCloud(errors.New("list failed")) },
			wantErr: `cannot resolve project_name "other-project"`,
		},
		{
			name:    "project name of another project unresolved during plan",
			config:  map[string]interface{}{"project_name": "other-project", "region_name": "test-region"},
			prepare: scopeCloudFailingPlanOnly,
			wantErr: fmt.Sprintf("cannot move from project %d to project %d", testProjectID, testOtherProjectID),
		},
	}

	for resourceName, base := range resources {
		resource := provider.Provider().ResourcesMap[resourceName]
		require.NotNil(t, resource)

		for _, tc := range cases {
			t.Run(resourceName+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				mc := tc.prepare()
				current := support.NewState(t, resource, cloud.Merge(base, cloud.WithProjectRegion(testProjectID, testRegionID), names), "test-id")

				state, diags := support.ApplyConfig(t, t.Context(), resource, current, cloud.Merge(base, tc.config), mc.TestMeta())

				support.RequireErrorDiagContains(t, diags, tc.wantErr)
				support.RequireStateID(t, state, "test-id")
				support.RequireStateAttrs(t, state, unchanged)
			})
		}
	}
}

func TestIntegrationCloudScopeNameOnUpdateKeepsUpdating(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_network"]
	require.NotNil(t, resource)

	base := cloud.Merge(cloud.WithName("test-network"), map[string]interface{}{"type": "vxlan"})

	cases := []struct {
		name   string
		state  map[string]interface{}
		config map[string]interface{}
		want   map[string]string
	}{
		{
			name: "project renamed",
			state: cloud.Merge(cloud.WithProjectRegion(testProjectID, testRegionID), map[string]interface{}{
				"project_name": "old-project",
			}),
			config: map[string]interface{}{"project_name": "test-project", "region_id": testRegionID},
			want:   map[string]string{"project_id": fmt.Sprint(testProjectID), "project_name": "test-project"},
		},
		{
			name:   "numeric project name",
			state:  cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{"project_name": fmt.Sprint(testProjectID), "region_id": testRegionID},
			want:   map[string]string{"project_id": fmt.Sprint(testProjectID), "project_name": fmt.Sprint(testProjectID)},
		},
		{
			name: "state without project id",
			state: map[string]interface{}{
				"project_name": "typo-project",
				"region_id":    testRegionID,
			},
			config: map[string]interface{}{"project_name": "test-project", "region_id": testRegionID},
			want:   map[string]string{"project_id": fmt.Sprint(testProjectID), "project_name": "test-project"},
		},
		{
			name:   "names after import",
			state:  cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{"project_name": "test-project", "region_name": "test-region"},
			want: map[string]string{
				"project_id":   fmt.Sprint(testProjectID),
				"project_name": "test-project",
				"region_id":    fmt.Sprint(testRegionID),
				"region_name":  "test-region",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mc := scopeResolvingCloud(nil)
			mc.Networks.On("Get", mock.Anything, testNetID).
				Return(sampleNetwork(testNetID, "test-network"), nil, nil).Once()
			current := support.NewState(t, resource, cloud.Merge(base, tc.state), testNetID)

			state, diags := support.ApplyConfig(t, t.Context(), resource, current, cloud.Merge(base, tc.config), mc.TestMeta())

			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testNetID)
			support.RequireStateAttrs(t, state, tc.want)
			mc.Networks.AssertExpectations(t)
		})
	}
}

func TestIntegrationCloudScopeNameOnUpdateIgnoresUnchangedName(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_network"]
	require.NotNil(t, resource)

	mc := scopeResolvingCloud(nil)
	mc.Networks.On("UpdateName", mock.Anything, testNetID,
		mock.MatchedBy(func(n *edgecloud.Name) bool {
			return n.Name == "renamed-network"
		}),
	).Return(nil, nil, nil).Once()
	mc.Networks.On("Get", mock.Anything, testNetID).
		Return(sampleNetwork(testNetID, "renamed-network"), nil, nil).Once()

	scope := map[string]interface{}{"project_name": "old-project", "region_id": testRegionID}
	current := support.NewState(t, resource, cloud.Merge(
		cloud.WithName("test-network"),
		map[string]interface{}{"type": "vxlan", "project_id": testProjectID},
		scope,
	), testNetID)

	state, diags := support.ApplyConfig(t, t.Context(), resource, current, cloud.Merge(
		cloud.WithName("renamed-network"),
		map[string]interface{}{"type": "vxlan"},
		scope,
	), mc.TestMeta())

	support.RequireNoDiags(t, diags)
	support.RequireStateAttrs(t, state, map[string]string{
		"name":         "renamed-network",
		"project_id":   fmt.Sprint(testProjectID),
		"project_name": "old-project",
	})
	mc.Networks.AssertExpectations(t)
}
