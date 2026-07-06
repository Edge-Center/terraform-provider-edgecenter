//go:build integration

package edgemon_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenteredgemon-go/checks"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkping"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckPingID = 102

func baseCheckPingConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":          "tf-ping",
		"enabled":       true,
		"place":         "country",
		"entities":      []interface{}{1, 2},
		"ip":            "1.1.1.1",
		"check_timeout": 5,
		"packet_size":   56,
		"count_packets": 4,
		"interval":      120,
		"retries":       3,
	}
}

func sampleCheckPing(name, place string, countPackets int, entities []int) *checkping.Response {
	return &checkping.Response{
		Name:         name,
		Enabled:      1,
		Place:        place,
		Entities:     entities,
		PacketSize:   56,
		CountPackets: countPackets,
		Interval:     120,
		CheckTimeout: 5,
		IP:           "1.1.1.1",
		Retries:      3,
	}
}

func checkPingCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checkping.Request) bool {
			return req.Name == "tf-ping" && req.IP == "1.1.1.1" &&
				req.Enabled == 1 && req.Place == "country" && req.CheckTimeout == 5
		}),
	).Return(&checks.CreateResponse{ID: testCheckPingID}, nil)

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(sampleCheckPing("tf-ping", "country", 4, []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckPingID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":          "tf-ping",
				"enabled":       "true",
				"place":         "country",
				"ip":            "1.1.1.1",
				"packet_size":   "56",
				"count_packets": "4",
				"entities.#":    "2",
				"entities.0":    "1",
				"entities.1":    "2",
			})
		},
	}
}

func checkPingCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckPingID}, nil)

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(sampleCheckPing("tf-ping", "all", 4, []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckPingConfig(), map[string]interface{}{
			"place":    "all",
			"entities": []interface{}{},
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"place":      "all",
				"entities.#": "0",
			})
		},
	}
}

func checkPingReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(sampleCheckPing("tf-ping", "country", 4, []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckPingID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":          "tf-ping",
				"ip":            "1.1.1.1",
				"count_packets": "4",
				"entities.#":    "2",
			})
		},
	}
}

func checkPingUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Update", mock.Anything, testCheckPingID,
		mock.MatchedBy(func(req *checkping.Request) bool {
			return req.CountPackets == 8
		}),
	).Return(nil)

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(sampleCheckPing("tf-ping", "country", 8, []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update count_packets",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		NewConfig: edgemon.Merge(baseCheckPingConfig(), map[string]interface{}{
			"count_packets": 8,
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"count_packets": "8",
			})
		},
	}
}

func checkPingCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkPingReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkPingDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Delete", mock.Anything, testCheckPingID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkPingDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Delete", mock.Anything, testCheckPingID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckPingID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkPingReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Get", mock.Anything, testCheckPingID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkPingDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckPing.On("Delete", mock.Anything, testCheckPingID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckPingID),
		CurrentState: baseCheckPingConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckPing_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_ping")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkPingCreateCase(),
		checkPingCreatePlaceAllCase(),
		checkPingReadCase(),
		checkPingUpdateCase(),
		checkPingCreateAPIFailureCase(),
		checkPingReadErrorCase(),
		checkPingDeleteCase(),
		checkPingDeleteAPIFailureCase(),
		checkPingReadNotFoundCase(),
		checkPingDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
