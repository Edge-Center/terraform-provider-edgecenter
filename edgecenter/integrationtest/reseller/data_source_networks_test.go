//go:build integration

package reseller_test

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	resellermock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/reseller/mock"
	resellersvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/reseller"
)

const (
	networksPlaceholderID = "-"
	networksListID        = "reseller_networks"

	testNetworkID             = "0f2a5b3c-1111-4000-8000-000000000001"
	testNetworkName           = "reseller-network"
	testNetworkType           = "vxlan"
	testNetworkMTU            = "1450"
	testNetworkRegionName     = "Luxembourg"
	testNetworkCreatedAt      = "2024-05-01T10:00:00+0000"
	testNetworkUpdatedAt      = "2024-05-02T11:00:00+0000"
	testNetworkCreatorTaskID  = "d6b6cf76-1111-4000-8000-000000000010"
	testNetworkTaskID         = "d6b6cf76-1111-4000-8000-000000000011"
	testNetworkSegmentationID = 1234
	testNetworkClientID       = 1188013
	testNetworkProjectID      = 777

	testSubnetID       = "9c3a1b2d-2222-4000-8000-000000000002"
	testSubnetName     = "reseller-subnet"
	testSubnetCIDR     = "192.168.10.0/24"
	testSubnetGateway  = "192.168.10.1"
	testSubnetAvailIPs = 250
	testSubnetTotalIPs = 256

	testHostRouteDestination = "10.100.0.0/24"
	testHostRouteNexthop     = "192.168.10.254"
)

var testHostRouteCIDR = mustParseCIDR(testHostRouteDestination)

func mustParseCIDR(value string) edgecloud.CIDR {
	parsed, err := edgecloud.ParseCIDRString(value)
	if err != nil {
		panic(err)
	}

	return *parsed
}

func sampleResellerSubnet() edgecloud.Subnetwork {
	return edgecloud.Subnetwork{
		ID:             testSubnetID,
		Name:           testSubnetName,
		EnableDHCP:     true,
		CIDR:           testSubnetCIDR,
		AvailableIps:   testSubnetAvailIPs,
		TotalIps:       testSubnetTotalIPs,
		HasRouter:      true,
		DNSNameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1")},
		HostRoutes: []edgecloud.HostRoute{
			{Destination: testHostRouteCIDR, NextHop: net.ParseIP(testHostRouteNexthop)},
		},
		GatewayIP: net.ParseIP(testSubnetGateway),
	}
}

func sampleResellerNetwork() edgecloud.ResellerNetwork {
	return edgecloud.ResellerNetwork{
		CreatedAt:      testNetworkCreatedAt,
		Default:        true,
		External:       true,
		Shared:         true,
		ID:             testNetworkID,
		MTU:            1450,
		Name:           testNetworkName,
		RegionID:       testRegionID,
		Region:         testNetworkRegionName,
		Type:           testNetworkType,
		Subnets:        []edgecloud.Subnetwork{sampleResellerSubnet()},
		CreatorTaskID:  testNetworkCreatorTaskID,
		TaskID:         testNetworkTaskID,
		SegmentationID: testNetworkSegmentationID,
		UpdatedAt:      testNetworkUpdatedAt,
		Metadata: []edgecloud.MetadataDetailed{
			{Key: "owner", Value: "reseller", ReadOnly: true},
		},
		ClientID:  testNetworkClientID,
		ProjectID: testNetworkProjectID,
	}
}

func sampleResellerNetworks() *edgecloud.ResellerNetworks {
	return &edgecloud.ResellerNetworks{
		Count:   1,
		Results: []edgecloud.ResellerNetwork{sampleResellerNetwork()},
	}
}

func setElementValues(state *terraform.InstanceState, prefix, field string) []string {
	values := make([]string, 0, 1)

	for key, value := range state.Attributes {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, "."+field) {
			values = append(values, value)
		}
	}

	return values
}

func networksFilterCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	var sent *edgecloud.ResellerNetworksListRequest

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(1).(*edgecloud.ResellerNetworksListRequest) }).
		Return(&edgecloud.ResellerNetworks{}, nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read sends every filter attribute to the api",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		CurrentState: map[string]interface{}{
			"network_type": testNetworkType,
			"order_by":     "name.asc",
			"shared":       true,
			"metadata_kv":  map[string]interface{}{"env": "prod", "tier": "gold"},
			"metadata_k":   []interface{}{"env", "tier"},
		},
		Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, testNetworkType, sent.NetworkType)
			require.Equal(t, "name.asc", sent.OrderBy)
			require.True(t, sent.Shared)
			require.JSONEq(t, `{"env":"prod","tier":"gold"}`, sent.MetadataKV)

			var keys []string
			require.NoError(t, json.Unmarshal([]byte(sent.MetadataK), &keys))
			require.ElementsMatch(t, []string{"env", "tier"}, keys)
		},
	}
}

func networksNoFilterCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	var sent *edgecloud.ResellerNetworksListRequest

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(1).(*edgecloud.ResellerNetworksListRequest) }).
		Return(&edgecloud.ResellerNetworks{}, nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read sends an empty request when the config carries no filter",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, edgecloud.ResellerNetworksListRequest{}, *sent)
		},
	}
}

func networksLiteralIDCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(sampleResellerNetworks(), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read stores the literal list name as the id",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, networksListID)
		},
	}
}

func networksEmptyListCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(&edgecloud.ResellerNetworks{}, nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read keeps the literal id and an empty list when the api returns nothing",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, networksListID)
			support.RequireStateAttrs(t, state, map[string]string{
				"networks.#": "0",
			})
		},
	}
}

func networksReadCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(sampleResellerNetworks(), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read maps a network onto the schema",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"networks.#":                      "1",
				"networks.0.created_at":           testNetworkCreatedAt,
				"networks.0.default":              "true",
				"networks.0.external":             "true",
				"networks.0.shared":               "true",
				"networks.0.id":                   testNetworkID,
				"networks.0.mtu":                  testNetworkMTU,
				"networks.0.name":                 testNetworkName,
				"networks.0.region_id":            testRegionIDStr,
				"networks.0.region_name":          testNetworkRegionName,
				"networks.0.type":                 testNetworkType,
				"networks.0.creator_task_id":      testNetworkCreatorTaskID,
				"networks.0.task_id":              testNetworkTaskID,
				"networks.0.segmentation_id":      fmt.Sprintf("%d", testNetworkSegmentationID),
				"networks.0.updated_at":           testNetworkUpdatedAt,
				"networks.0.client_id":            fmt.Sprintf("%d", testNetworkClientID),
				"networks.0.metadata.#":           "1",
				"networks.0.metadata.0.key":       "owner",
				"networks.0.metadata.0.value":     "reseller",
				"networks.0.metadata.0.read_only": "true",
			})
		},
	}
}

func networksSubnetCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(sampleResellerNetworks(), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read maps the nested subnet block onto the schema",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"networks.0.subnets.#":                   "1",
				"networks.0.subnets.0.id":                testSubnetID,
				"networks.0.subnets.0.name":              testSubnetName,
				"networks.0.subnets.0.available_ips":     fmt.Sprintf("%d", testSubnetAvailIPs),
				"networks.0.subnets.0.total_ips":         fmt.Sprintf("%d", testSubnetTotalIPs),
				"networks.0.subnets.0.enable_dhcp":       "true",
				"networks.0.subnets.0.has_router":        "true",
				"networks.0.subnets.0.cidr":              testSubnetCIDR,
				"networks.0.subnets.0.gateway_ip":        testSubnetGateway,
				"networks.0.subnets.0.dns_nameservers.#": "2",
				"networks.0.subnets.0.dns_nameservers.0": "8.8.8.8",
				"networks.0.subnets.0.dns_nameservers.1": "1.1.1.1",
				"networks.0.subnets.0.host_routes.#":     "1",
			})

			hostRoutes := "networks.0.subnets.0.host_routes."
			require.Equal(t, []string{testHostRouteDestination}, setElementValues(state, hostRoutes, "destination"))
			require.Equal(t, []string{testHostRouteNexthop}, setElementValues(state, hostRoutes, "nexthop"))
		},
	}
}

func networksProjectIDHoldsRegionIDCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(sampleResellerNetworks(), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read fills project_id with the region id because the mapper never reads the project field",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"networks.0.project_id": testRegionIDStr,
				"networks.0.region_id":  testRegionIDStr,
			})
			require.NotEqual(t, fmt.Sprintf("%d", testNetworkProjectID), state.Attributes["networks.0.project_id"])
		},
	}
}

func networksAPIFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Networks.On("List", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: reseller api key is required"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read surfaces the api error and leaves the id untouched",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: networksPlaceholderID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "reseller api key is required")
			support.RequireStateID(t, state, networksPlaceholderID)
		},
	}
}

func TestIntegrationResellerNetworks_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := resellerDataSource(t, resellersvc.ResellerNetworksDataSource)

	cases := []support.ResourceCase[*resellermock.MockedReseller]{
		networksFilterCase(),
		networksNoFilterCase(),
		networksLiteralIDCase(),
		networksEmptyListCase(),
		networksReadCase(),
		networksSubnetCase(),
		networksProjectIDHoldsRegionIDCase(),
		networksAPIFailureCase(),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*resellermock.MockedReseller])
}
