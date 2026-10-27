//go:build integration

package edgecenter_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testOtherProjectID = 2
	testOtherRegionID  = 2
)

func scopeResolvingCloud(listErr error) *cloudmock.MockedCloud {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{
			{ID: testProjectID, Name: "test-project"},
			{ID: testOtherProjectID, Name: "other-project"},
			{ID: 3, Name: "shared-name"},
			{ID: 4, Name: "shared-name"},
		}, nil, listErr).Maybe()
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{
			{ID: testRegionID, DisplayName: "test-region"},
			{ID: testOtherRegionID, DisplayName: "other-region"},
		}, nil, listErr).Maybe()

	return mc
}

func TestIntegrationCloudScopeChange(t *testing.T) {
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

	projectMove := fmt.Sprintf("cannot move from project %d to project %d", testProjectID, testOtherProjectID)
	regionMove := fmt.Sprintf("cannot move from region %d to region %d", testRegionID, testOtherRegionID)

	cases := []struct {
		name    string
		state   map[string]interface{}
		config  map[string]interface{}
		listErr error
		wantErr string
	}{
		{
			name:   "same ids",
			state:  cloud.WithProjectRegion(testProjectID, testRegionID),
			config: cloud.WithProjectRegion(testProjectID, testRegionID),
		},
		{
			name:    "new region id",
			state:   cloud.WithProjectRegion(testProjectID, testRegionID),
			config:  cloud.WithProjectRegion(testProjectID, testOtherRegionID),
			wantErr: regionMove,
		},
		{
			name:    "new project id",
			state:   cloud.WithProjectRegion(testProjectID, testRegionID),
			config:  cloud.WithProjectRegion(testOtherProjectID, testRegionID),
			wantErr: projectMove,
		},
		{
			name:  "imported state with names of the same scope",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "test-project",
				"region_name":  "test-region",
			},
		},
		{
			name: "names of the same scope",
			state: cloud.Merge(cloud.WithProjectRegion(testProjectID, testRegionID), map[string]interface{}{
				"project_name": "test-project",
				"region_name":  "test-region",
			}),
			config: map[string]interface{}{
				"project_name": "test-project",
				"region_name":  "test-region",
			},
		},
		{
			name: "region name of another region",
			state: cloud.Merge(cloud.WithProjectRegion(testProjectID, testRegionID), map[string]interface{}{
				"project_name": "test-project",
				"region_name":  "test-region",
			}),
			config: map[string]interface{}{
				"project_name": "test-project",
				"region_name":  "other-region",
			},
			wantErr: regionMove,
		},
		{
			name:  "project name of another project",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "other-project",
				"region_id":    testRegionID,
			},
			wantErr: projectMove,
		},
		{
			name:  "numeric project name of the same project",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "1",
				"region_id":    testRegionID,
			},
		},
		{
			name:  "numeric project name of another project",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "2",
				"region_id":    testRegionID,
			},
			wantErr: projectMove,
		},
		{
			name:  "numeric region name of another region",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_id":  testProjectID,
				"region_name": "2",
			},
			wantErr: regionMove,
		},
		{
			name: "project name that the same apply creates",
			state: cloud.Merge(cloud.WithProjectRegion(testProjectID, testRegionID), map[string]interface{}{
				"project_name": "test-project",
			}),
			config: map[string]interface{}{
				"project_name": "renamed-project",
				"region_id":    testRegionID,
			},
		},
		{
			name:  "project name of another project while the lists are unavailable",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "other-project",
				"region_name":  "other-region",
			},
			listErr: errors.New("list failed"),
		},
		{
			name:  "project name shared by two projects",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "shared-name",
				"region_id":    testRegionID,
			},
		},
	}

	for resourceName, base := range resources {
		resource := provider.Provider().ResourcesMap[resourceName]
		require.NotNil(t, resource)

		for _, tc := range cases {
			t.Run(resourceName+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				mc := scopeResolvingCloud(tc.listErr)
				state := support.NewState(t, resource, cloud.Merge(base, tc.state), "test-id")
				config := terraform.NewResourceConfigRaw(cloud.Merge(base, tc.config))

				diff, err := resource.Diff(t.Context(), state, config, mc.TestMeta())
				if tc.wantErr != "" {
					require.ErrorContains(t, err, tc.wantErr)

					return
				}

				require.NoError(t, err)
				if diff != nil {
					require.False(t, diff.RequiresNew(), "unexpected replacement: %#v", diff.Attributes)
				}
			})
		}
	}
}
