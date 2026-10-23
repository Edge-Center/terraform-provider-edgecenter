//go:build integration

package edgecenter_test

import (
	"strconv"
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

func scopeResolvingCloud() *cloudmock.MockedCloud {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{
			{ID: testProjectID, Name: "test-project"},
			{ID: testOtherProjectID, Name: "other-project"},
			{ID: 3, Name: "shared-name"},
			{ID: 4, Name: "shared-name"},
		}, nil, nil).Maybe()
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{
			{ID: testRegionID, DisplayName: "test-region"},
			{ID: testOtherRegionID, DisplayName: "other-region"},
		}, nil, nil).Maybe()

	return mc
}

func TestIntegrationCloudScopeForceNew(t *testing.T) {
	t.Parallel()

	resources := map[string]map[string]interface{}{
		"edgecenter_volume": cloud.Merge(
			cloud.WithName("test-volume"),
			cloud.WithSize(1),
			cloud.WithTypeName("standard"),
		),
		"edgecenter_network": cloud.Merge(
			cloud.WithName("test-network"),
			map[string]interface{}{"type": "vxlan"},
		),
		"edgecenter_securitygroup": cloud.WithName("test-sg"),
	}

	cases := []struct {
		name        string
		state       map[string]interface{}
		config      map[string]interface{}
		requiresNew bool
		projectID   int
		regionID    int
	}{
		{
			name:   "same ids",
			state:  cloud.WithProjectRegion(testProjectID, testRegionID),
			config: cloud.WithProjectRegion(testProjectID, testRegionID),
		},
		{
			name:        "new region id",
			state:       cloud.WithProjectRegion(testProjectID, testRegionID),
			config:      cloud.WithProjectRegion(testProjectID, testOtherRegionID),
			requiresNew: true,
			regionID:    testOtherRegionID,
		},
		{
			name:        "new project id",
			state:       cloud.WithProjectRegion(testProjectID, testRegionID),
			config:      cloud.WithProjectRegion(testOtherProjectID, testRegionID),
			requiresNew: true,
			projectID:   testOtherProjectID,
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
			requiresNew: true,
			regionID:    testOtherRegionID,
		},
		{
			name:  "project name of another project",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_name": "other-project",
				"region_id":    testRegionID,
			},
			requiresNew: true,
			projectID:   testOtherProjectID,
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
			requiresNew: true,
			projectID:   testOtherProjectID,
		},
		{
			name:  "numeric region name of another region",
			state: cloud.WithProjectRegion(testProjectID, testRegionID),
			config: map[string]interface{}{
				"project_id":  testProjectID,
				"region_name": "2",
			},
			requiresNew: true,
			regionID:    testOtherRegionID,
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

				mc := scopeResolvingCloud()
				state := support.NewState(t, resource, cloud.Merge(base, tc.state), "test-id")
				config := terraform.NewResourceConfigRaw(cloud.Merge(base, tc.config))

				diff, err := resource.Diff(t.Context(), state, config, mc.TestMeta())
				require.NoError(t, err)

				if !tc.requiresNew {
					if diff != nil {
						require.False(t, diff.RequiresNew(), "unexpected replacement: %#v", diff.Attributes)
					}

					return
				}

				require.NotNil(t, diff)
				require.True(t, diff.RequiresNew(), "expected replacement: %#v", diff.Attributes)
				requirePlannedScope(t, diff, "project_id", tc.projectID)
				requirePlannedScope(t, diff, "region_id", tc.regionID)
			})
		}
	}
}

func requirePlannedScope(t *testing.T, diff *terraform.InstanceDiff, key string, want int) {
	t.Helper()

	if want == 0 {
		return
	}

	attr := diff.Attributes[key]
	require.NotNil(t, attr)
	require.True(t, attr.RequiresNew, "%s must force replacement", key)
	if attr.NewComputed {
		return
	}

	planned, err := strconv.Atoi(attr.New)
	require.NoError(t, err)
	require.Equal(t, want, planned)
}
