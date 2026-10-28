//go:build integration

package edgecenter_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func TestIntegrationCloudScopeSchema(t *testing.T) {
	t.Parallel()

	resources := []struct {
		name     string
		forceNew bool
	}{
		{name: "edgecenter_servergroup", forceNew: true},
		{name: "edgecenter_volume"},
		{name: "edgecenter_loadbalancer", forceNew: true},
		{name: "edgecenter_loadbalancerv2", forceNew: true},
		{name: "edgecenter_floatingip"},
		{name: "edgecenter_network"},
		{name: "edgecenter_reservedfixedip"},
		{name: "edgecenter_router"},
		{name: "edgecenter_lifecyclepolicy", forceNew: true},
	}
	pairs := [][2]string{
		{"project_id", "project_name"},
		{"region_id", "region_name"},
	}
	resourceMap := provider.Provider().ResourcesMap

	for _, tc := range resources {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := resourceMap[tc.name]
			require.NotNil(t, resource)
			for _, pair := range pairs {
				for _, key := range pair {
					field := resource.Schema[key]
					require.NotNilf(t, field, "attribute %q is missing", key)
					require.Truef(t, field.Optional, "attribute %q must be optional", key)
					require.Truef(t, field.Computed, "attribute %q must be computed", key)
					require.Equalf(t, []string{pair[0], pair[1]}, field.ExactlyOneOf,
						"attribute %q must remain exactly one of the scope pair", key)
					require.Equalf(t, tc.forceNew, field.ForceNew, "attribute %q force new flag", key)
				}
			}
		})
	}
}

func TestIntegrationCloudNamedScopeDoesNotDrift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		config        map[string]interface{}
		computedState map[string]interface{}
	}{
		{
			name: "edgecenter_volume",
			config: cloud.Merge(
				cloud.WithName("test-volume"),
				cloud.WithSize(1),
				cloud.WithTypeName("standard"),
			),
			computedState: map[string]interface{}{
				"metadata_read_only": []interface{}{
					map[string]interface{}{"key": "source", "value": "test", "read_only": true},
				},
			},
		},
		{
			name: "edgecenter_servergroup",
			config: cloud.Merge(
				cloud.WithName("test-servergroup"),
				map[string]interface{}{"policy": "anti-affinity"},
			),
			computedState: map[string]interface{}{
				"instances": []interface{}{
					map[string]interface{}{"instance_id": "instance-id", "instance_name": "test-instance"},
				},
			},
		},
	}
	resourceMap := provider.Provider().ResourcesMap

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := resourceMap[tc.name]
			require.NotNil(t, resource)
			configs := []struct {
				name  string
				scope map[string]interface{}
			}{
				{
					name: "names",
					scope: map[string]interface{}{
						"project_name": "test-project",
						"region_name":  "test-region",
					},
				},
				{name: "ids", scope: cloud.WithProjectRegion(testProjectID, testRegionID)},
			}
			for _, scope := range configs {
				t.Run(scope.name, func(t *testing.T) {
					t.Parallel()

					config := cloud.Merge(tc.config, scope.scope)
					afterRead := cloud.Merge(config,
						cloud.WithProjectRegion(testProjectID, testRegionID), tc.computedState)
					state := support.NewState(t, resource, afterRead, "test-id")
					diff, err := resource.Diff(t.Context(), state, terraform.NewResourceConfigRaw(config), nil)
					require.NoError(t, err)
					if diff != nil {
						require.True(t, diff.Empty(), "the refreshed state must not produce another plan: %#v", diff.Attributes)
					}
				})
			}
		})
	}
}
