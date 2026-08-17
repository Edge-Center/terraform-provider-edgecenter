//go:build integration

package mkaas_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	mkaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/mkaas"
)

func planDiff(
	t *testing.T,
	resource *schema.Resource,
	current *terraform.InstanceState,
	config map[string]interface{},
) (*terraform.InstanceDiff, error) {
	t.Helper()

	return resource.Diff(t.Context(), current, terraform.NewResourceConfigRaw(config), nil)
}

func requirePlanDiff(
	t *testing.T,
	resource *schema.Resource,
	current *terraform.InstanceState,
	config map[string]interface{},
) *terraform.InstanceDiff {
	t.Helper()

	diff, err := planDiff(t, resource, current, config)
	require.NoError(t, err)

	return diff
}

func importedState(t *testing.T, resource *schema.Resource, importID string) *terraform.InstanceState {
	t.Helper()

	data := resource.Data(nil)
	data.SetId(importID)

	results, err := resource.Importer.StateContext(t.Context(), data, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)

	return results[0].State()
}

func TestIntegrationMKaaSClusterDiff_ReplacementCensus(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)
	current := support.NewState(t, resource, clusterState(), testClusterIDStr)

	inPlace := map[string]map[string]interface{}{
		"network_id":                   {"network_id": "9f9f9f9f-1111-2222-3333-444444444444"},
		"subnet_id":                    {"subnet_id": "8e8e8e8e-1111-2222-3333-444444444444"},
		"ssh_keypair_name":             {"ssh_keypair_name": "another-key"},
		"publish_kube_api_to_internet": {"publish_kube_api_to_internet": true},
		"control_plane.0.flavor":       {"control_plane": controlPlane(map[string]interface{}{"flavor": "g1-standard-4-8"})},
		"control_plane.0.volume_size":  {"control_plane": controlPlane(map[string]interface{}{"volume_size": 100})},
	}

	for name, change := range inPlace {
		t.Run("changing "+name+" plans an in place update the update guard refuses", func(t *testing.T) {
			t.Parallel()

			diff := requirePlanDiff(t, resource, current, clusterConfig(change))

			require.NotNil(t, diff)
			require.False(t, diff.RequiresNew(),
				"the api cannot patch this attribute, so the plan should replace the cluster")
		})
	}

	replaced := map[string]map[string]interface{}{
		"pod_subnet":     {"pod_subnet": "10.110.0.0/16"},
		"service_subnet": {"service_subnet": "10.210.0.0/16"},
	}

	for name, change := range replaced {
		t.Run("changing "+name+" replaces the cluster", func(t *testing.T) {
			t.Parallel()

			diff := requirePlanDiff(t, resource, current, clusterConfig(change))

			require.NotNil(t, diff)
			require.True(t, diff.RequiresNew())
		})
	}
}

func TestIntegrationMKaaSClusterDiff_ScopeAndImport(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	t.Run("a name based config diffs the ids the read resolved back to zero", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, clusterState(), testClusterIDStr)
		config := merge(clusterConfig(), map[string]interface{}{
			"project_id":   nil,
			"region_id":    nil,
			"project_name": "test-project",
			"region_name":  "test-region",
		})

		diff := requirePlanDiff(t, resource, current, config)

		require.NotNil(t, diff)
		require.Contains(t, diff.Attributes, "project_id")
		require.Contains(t, diff.Attributes, "region_id")
		require.Equal(t, "0", diff.Attributes["project_id"].New)
		require.Equal(t, "0", diff.Attributes["region_id"].New)
		require.False(t, diff.RequiresNew(),
			"the resulting in place update is exactly what the update guard rejects")
	})

	t.Run("the importer splits project, region and cluster id", func(t *testing.T) {
		t.Parallel()

		state := importedState(t, resource, "1:8:42")

		require.Equal(t, testClusterIDStr, state.ID)
		require.Equal(t, testProjectIDStr, state.Attributes["project_id"])
		require.Equal(t, testRegionIDStr, state.Attributes["region_id"])
	})

	t.Run("the importer rejects a malformed id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId("1:8")

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.ErrorContains(t, err, "importing MKaaS cluster: failed import: wrong input id")
	})

	t.Run("an imported cluster plans a publish flag change the update refuses", func(t *testing.T) {
		t.Parallel()

		state := importedState(t, resource, "1:8:42")
		for key, value := range map[string]string{
			"name":                        testClusterName,
			"ssh_keypair_name":            testKeypairName,
			"network_id":                  testNetworkID,
			"subnet_id":                   testSubnetID,
			"pod_subnet":                  testPodSubnet,
			"service_subnet":              testServiceSubnet,
			"control_plane.#":             "1",
			"control_plane.0.flavor":      testFlavor,
			"control_plane.0.node_count":  "1",
			"control_plane.0.volume_size": "50",
			"control_plane.0.volume_type": "ssd_hiiops",
			"control_plane.0.version":     testShortVersion,
		} {
			state.Attributes[key] = value
		}

		diff := requirePlanDiff(t, resource, state,
			clusterConfig(map[string]interface{}{"publish_kube_api_to_internet": true}))

		require.NotNil(t, diff)
		require.Contains(t, diff.Attributes, "publish_kube_api_to_internet",
			"read never writes the flag back, so an imported cluster always plans this change")
		require.False(t, diff.RequiresNew())
	})
}

func TestIntegrationMKaaSClusterDiff_SubnetValidation(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	cases := []struct {
		name    string
		config  map[string]interface{}
		message string
	}{
		{
			name: "identical pod and service subnets are rejected",
			config: clusterConfig(map[string]interface{}{
				"service_subnet": testPodSubnet,
			}),
			message: "pod_subnet and service_subnet cannot be the same CIDR",
		},
		{
			name: "overlapping subnets are rejected",
			config: clusterConfig(map[string]interface{}{
				"pod_subnet":     "10.0.0.0/8",
				"service_subnet": "10.200.0.0/16",
			}),
			message: "intersects with service_subnet",
		},
		{
			name: "a malformed pod subnet is rejected",
			config: clusterConfig(map[string]interface{}{
				"pod_subnet": "not-a-cidr",
			}),
			message: "invalid pod_subnet",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := planDiff(t, resource, nil, tc.config)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.message)
		})
	}

	t.Run("a range that only starts inside rfc1918 is accepted", func(t *testing.T) {
		t.Parallel()

		diff, err := planDiff(t, resource, nil, clusterConfig(map[string]interface{}{
			"pod_subnet":     "192.168.0.0/15",
			"service_subnet": "10.200.0.0/16",
		}))

		require.NoError(t, err,
			"containment is decided by the network address alone, so half of 192.168.0.0/15 is public")
		require.NotNil(t, diff)
	})
}

func TestIntegrationMKaaSPoolDiff_ReplacementCensus(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)
	current := support.NewState(t, resource, poolState(), testPoolIDStr)

	inPlace := map[string]map[string]interface{}{
		"flavor":      {"flavor": "g1-standard-4-8"},
		"volume_size": {"volume_size": 40},
		"volume_type": {"volume_type": "ssd_hiiops"},
	}

	for name, change := range inPlace {
		t.Run("changing "+name+" plans an in place update the update guard refuses", func(t *testing.T) {
			t.Parallel()

			diff := requirePlanDiff(t, resource, current, poolConfig(change))

			require.NotNil(t, diff)
			require.False(t, diff.RequiresNew(),
				"the api cannot patch this attribute, so the plan should replace the pool")
		})
	}

	t.Run("changing cluster_id replaces the pool", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t, resource, current, poolConfig(map[string]interface{}{"cluster_id": 99}))

		require.NotNil(t, diff)
		require.True(t, diff.RequiresNew())
	})

	t.Run("reordering security_group_ids is a change because the attribute is an ordered list", func(t *testing.T) {
		t.Parallel()

		state := support.NewState(t, resource, poolState(map[string]interface{}{
			"security_group_ids": []interface{}{"sg-1", "sg-2"},
		}), testPoolIDStr)

		diff := requirePlanDiff(t, resource, state, poolConfig(map[string]interface{}{
			"security_group_ids": []interface{}{"sg-2", "sg-1"},
		}))

		require.NotNil(t, diff)
		require.Contains(t, diff.Attributes, "security_group_ids.0")
	})
}

func TestIntegrationMKaaSPoolDiff_CustomizeDiff(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	t.Run("duplicate taint keys are rejected", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, poolConfig(map[string]interface{}{
			"taints": []interface{}{
				taint("dedicated", "gpu", "NoSchedule"),
				taint("dedicated", "cpu", "NoExecute"),
			},
		}))

		require.Error(t, err)
		require.Contains(t, err.Error(), `duplicate taint key "dedicated"`)
	})

	t.Run("an inverted autoscale range is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, merge(poolConfig(), map[string]interface{}{
			"node_count":   nil,
			"scale_policy": scalePolicy(5, 2),
		}))

		require.Error(t, err)
		require.Contains(t, err.Error(), "must be >=")
	})

	t.Run("a node count change marks the live count unknown", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, poolState(), testPoolIDStr)

		diff := requirePlanDiff(t, resource, current, poolConfig(map[string]interface{}{"node_count": 5}))

		require.NotNil(t, diff)
		require.Contains(t, diff.Attributes, "current_node_count")
		require.True(t, diff.Attributes["current_node_count"].NewComputed)
	})
}

func TestIntegrationMKaaSPoolDiff_Import(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	t.Run("the importer takes project, region, pool and cluster in that order", func(t *testing.T) {
		t.Parallel()

		state := importedState(t, resource, "1:8:7:42")

		require.Equal(t, testPoolIDStr, state.ID)
		require.Equal(t, testProjectIDStr, state.Attributes["project_id"])
		require.Equal(t, testRegionIDStr, state.Attributes["region_id"])
		require.Equal(t, testClusterIDStr, state.Attributes["cluster_id"])
	})

	t.Run("the importer rejects a three segment id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId("1:8:7")

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.ErrorContains(t, err, "importing MKaaS pool: failed import: wrong input id")
	})

	t.Run("the importer rejects a non numeric cluster id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId("1:8:7:not-a-number")

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.ErrorContains(t, err, "invalid cluster_id")
	})
}
