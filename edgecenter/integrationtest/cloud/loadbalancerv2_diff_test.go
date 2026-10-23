//go:build integration

package edgecenter_test

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const unknownConfigValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

func TestIntegrationLoadBalancerV2Diff_VipPortID(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_loadbalancerv2"]
	base := cloud.Merge(
		cloud.WithProjectRegion(testProjectID, testRegionID),
		map[string]interface{}{"name": "test-lb", "flavor": "lb1-1-2"},
	)

	cases := []struct {
		name        string
		vipPortID   string
		requiresNew bool
	}{
		{name: "same port", vipPortID: "port-old"},
		{name: "known new port", vipPortID: "port-new", requiresNew: true},
		{name: "port of a reservation created in the same apply", vipPortID: unknownConfigValue, requiresNew: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := support.NewState(t, resource, cloud.Merge(base, map[string]interface{}{
				"vip_port_id": "port-old",
				"vip_address": "203.0.113.10",
			}), testLoadBalancerID)

			rawConfig := support.NewRawConfig(t, resource, cloud.Merge(base, map[string]interface{}{"vip_port_id": "placeholder"})).AsValueMap()
			rawConfig["vip_port_id"] = cty.StringVal(tc.vipPortID)
			if tc.vipPortID == unknownConfigValue {
				rawConfig["vip_port_id"] = cty.UnknownVal(cty.String)
			}
			state.RawConfig = cty.ObjectVal(rawConfig)

			config := terraform.NewResourceConfigRaw(cloud.Merge(base, map[string]interface{}{"vip_port_id": tc.vipPortID}))
			diff, err := resource.Diff(t.Context(), state, config, nil)
			require.NoError(t, err)

			if !tc.requiresNew {
				require.True(t, diff == nil || !diff.RequiresNew(), "unexpected replacement: %#v", diff)
				return
			}

			require.NotNil(t, diff)
			require.True(t, diff.RequiresNew(), "expected replacement: %#v", diff.Attributes)
		})
	}
}

func TestIntegrationLoadBalancerV2Diff_Flavor(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_loadbalancerv2"]
	base := cloud.Merge(
		cloud.WithProjectRegion(testProjectID, testRegionID),
		map[string]interface{}{"name": "test-lb", "vip_port_id": "port-old"},
	)

	cases := []struct {
		name        string
		flavor      string
		requiresNew bool
	}{
		{name: "same flavor", flavor: "lb1-1-2"},
		{name: "known new flavor", flavor: "lb1-2-4", requiresNew: true},
		{name: "flavor unknown during plan", flavor: unknownConfigValue, requiresNew: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := support.NewState(t, resource, cloud.Merge(base, map[string]interface{}{
				"flavor":      "lb1-1-2",
				"vip_address": "203.0.113.10",
			}), testLoadBalancerID)

			rawConfig := support.NewRawConfig(t, resource, cloud.Merge(base, map[string]interface{}{"flavor": "placeholder"})).AsValueMap()
			rawConfig["flavor"] = cty.StringVal(tc.flavor)
			if tc.flavor == unknownConfigValue {
				rawConfig["flavor"] = cty.UnknownVal(cty.String)
			}
			state.RawConfig = cty.ObjectVal(rawConfig)

			config := terraform.NewResourceConfigRaw(cloud.Merge(base, map[string]interface{}{"flavor": tc.flavor}))
			diff, err := resource.Diff(t.Context(), state, config, nil)
			require.NoError(t, err)

			if !tc.requiresNew {
				require.True(t, diff == nil || !diff.RequiresNew(), "unexpected replacement: %#v", diff)
				return
			}

			require.NotNil(t, diff)
			require.True(t, diff.RequiresNew(), "expected replacement: %#v", diff.Attributes)
		})
	}
}
