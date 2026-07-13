//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/shielding"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testShieldingResourceID = 123
	testShieldingPop        = 5
)

func shieldingPopPtr(v int) *int { return &v }

func shieldingConfig(pop int) map[string]interface{} {
	return map[string]interface{}{
		"resource_id":   testShieldingResourceID,
		"shielding_pop": pop,
	}
}

func shieldingCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Update", mock.Anything, int64(testShieldingResourceID),
		mock.MatchedBy(func(req *shielding.UpdateShieldingData) bool {
			return req.ShieldingPop != nil && *req.ShieldingPop == testShieldingPop
		}),
	).Return(&shielding.ShieldingData{ShieldingPop: shieldingPopPtr(testShieldingPop)}, nil)

	mc.Shielding.On("Get", mock.Anything, int64(testShieldingResourceID)).
		Return(&shielding.ShieldingData{ShieldingPop: shieldingPopPtr(testShieldingPop)}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: shieldingConfig(testShieldingPop),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testShieldingResourceID))
			support.RequireStateAttrs(t, state, map[string]string{
				"resource_id":   fmt.Sprintf("%d", testShieldingResourceID),
				"shielding_pop": fmt.Sprintf("%d", testShieldingPop),
			})
		},
	}
}

// The importer is ImportStatePassthroughContext, so on `terraform import` the state
// carries only the ID and Read is the only thing that can populate resource_id.
// Seeding a different resource_id is what makes the assertion able to fail.
func shieldingReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Get", mock.Anything, int64(testShieldingResourceID)).
		Return(&shielding.ShieldingData{ShieldingPop: shieldingPopPtr(7)}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "read derives resource_id from state id",
		Op:        support.OpRead,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		CurrentID: fmt.Sprintf("%d", testShieldingResourceID),
		CurrentState: map[string]interface{}{
			"resource_id":   999,
			"shielding_pop": testShieldingPop,
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"resource_id":   fmt.Sprintf("%d", testShieldingResourceID),
				"shielding_pop": "7",
			})
		},
	}
}

func shieldingUpdateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Update", mock.Anything, int64(testShieldingResourceID),
		mock.MatchedBy(func(req *shielding.UpdateShieldingData) bool {
			return req.ShieldingPop != nil && *req.ShieldingPop == 9
		}),
	).Return(&shielding.ShieldingData{ShieldingPop: shieldingPopPtr(9)}, nil)

	mc.Shielding.On("Get", mock.Anything, int64(testShieldingResourceID)).
		Return(&shielding.ShieldingData{ShieldingPop: shieldingPopPtr(9)}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update shielding pop",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testShieldingResourceID),
		CurrentState: shieldingConfig(testShieldingPop),
		NewConfig:    shieldingConfig(9),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"shielding_pop": "9",
			})
		},
	}
}

func shieldingDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Update", mock.Anything, int64(testShieldingResourceID),
		mock.MatchedBy(func(req *shielding.UpdateShieldingData) bool {
			return req.ShieldingPop == nil
		}),
	).Return(&shielding.ShieldingData{}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete resets shielding pop to null",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testShieldingResourceID),
		CurrentState: shieldingConfig(testShieldingPop),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func shieldingCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Update", mock.Anything, int64(testShieldingResourceID), mock.Anything).
		Return(nil, fmt.Errorf("api error: unknown shielding location"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: shieldingConfig(testShieldingPop),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "unknown shielding location")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func shieldingReadInvalidIDCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read rejects non numeric id",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "not-a-number",
		CurrentState: shieldingConfig(testShieldingPop),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "provided wrong resource_id")
		},
	}
}

func shieldingReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Shielding.On("Get", mock.Anything, int64(testShieldingResourceID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testShieldingResourceID),
		CurrentState: shieldingConfig(testShieldingPop),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func TestIntegrationShielding_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_shielding")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		shieldingCreateCase(),
		shieldingReadCase(),
		shieldingUpdateCase(),
		shieldingDeleteCase(),
		shieldingCreateAPIFailureCase(),
		shieldingReadInvalidIDCase(),
		shieldingReadAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
