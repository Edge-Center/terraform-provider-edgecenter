//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/shielding"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testDatacenter         = "dc-1"
	testShieldingLocID     = 17
	testOtherShieldingLoc  = 18
	testOtherDatacenterKey = "dc-2"
)

func shieldingLocations() *[]shielding.ShieldingLocations {
	locations := []shielding.ShieldingLocations{
		{ID: testOtherShieldingLoc, Datacenter: testOtherDatacenterKey, Country: "DE", City: "Frankfurt"},
		{ID: testShieldingLocID, Datacenter: testDatacenter, Country: "NL", City: "Amsterdam"},
	}

	return &locations
}

func shieldingLocationReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("GetShieldingLocations", mock.Anything).Return(shieldingLocations(), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "resolves datacenter to location id",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{"datacenter": testDatacenter},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testShieldingLocID))
			support.RequireStateAttrs(t, state, map[string]string{
				"datacenter": testDatacenter,
			})
		},
	}
}

func shieldingLocationUnknownDatacenterCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("GetShieldingLocations", mock.Anything).Return(shieldingLocations(), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "unknown datacenter is an error",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{"datacenter": "dc-missing"},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "shielding location for datacenter dc-missing not found")
		},
	}
}

func shieldingLocationAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("GetShieldingLocations", mock.Anything).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{"datacenter": testDatacenter},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func TestIntegrationShieldingLocation_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := cdnDataSource(t, "edgecenter_cdn_shielding_location")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		shieldingLocationReadCase(),
		shieldingLocationUnknownDatacenterCase(),
		shieldingLocationAPIFailureCase(),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
