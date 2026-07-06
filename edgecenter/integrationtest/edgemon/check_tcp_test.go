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
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checktcp"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckTCPID = 103

func baseCheckTCPConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":          "tf-tcp",
		"enabled":       true,
		"place":         "country",
		"priority":      "warning",
		"entities":      []interface{}{1, 2},
		"ip":            "1.1.1.1",
		"port":          8080,
		"interval":      120,
		"check_timeout": 2,
		"retries":       3,
	}
}

func sampleCheckTCP(name, place, priority string, entities []int) *checktcp.Response {
	return &checktcp.Response{
		Name:         name,
		Enabled:      1,
		Place:        place,
		Entities:     entities,
		Priority:     priority,
		Interval:     120,
		CheckTimeout: 2,
		IP:           "1.1.1.1",
		Port:         8080,
		Retries:      3,
	}
}

func checkTCPCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checktcp.Request) bool {
			return req.Name == "tf-tcp" && req.IP == "1.1.1.1" &&
				req.Port == 8080 && req.Priority == "warning" &&
				req.Enabled == 1 && req.Place == "country"
		}),
	).Return(&checks.CreateResponse{ID: testCheckTCPID}, nil)

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(sampleCheckTCP("tf-tcp", "country", "warning", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckTCPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-tcp",
				"enabled":    "true",
				"place":      "country",
				"priority":   "warning",
				"ip":         "1.1.1.1",
				"port":       "8080",
				"entities.#": "2",
				"entities.0": "1",
				"entities.1": "2",
			})
		},
	}
}

func checkTCPCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckTCPID}, nil)

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(sampleCheckTCP("tf-tcp", "all", "warning", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckTCPConfig(), map[string]interface{}{
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

func checkTCPReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(sampleCheckTCP("tf-tcp", "country", "warning", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckTCPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-tcp",
				"priority":   "warning",
				"port":       "8080",
				"entities.#": "2",
			})
		},
	}
}

func checkTCPUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Update", mock.Anything, testCheckTCPID,
		mock.MatchedBy(func(req *checktcp.Request) bool {
			return req.Priority == "critical"
		}),
	).Return(nil)

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(sampleCheckTCP("tf-tcp", "country", "critical", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update priority",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		NewConfig: edgemon.Merge(baseCheckTCPConfig(), map[string]interface{}{
			"priority": "critical",
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"priority": "critical",
			})
		},
	}
}

func checkTCPCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkTCPReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkTCPDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Delete", mock.Anything, testCheckTCPID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkTCPDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Delete", mock.Anything, testCheckTCPID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckTCPID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkTCPReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Get", mock.Anything, testCheckTCPID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkTCPDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckTCP.On("Delete", mock.Anything, testCheckTCPID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckTCPID),
		CurrentState: baseCheckTCPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckTCP_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_tcp")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkTCPCreateCase(),
		checkTCPCreatePlaceAllCase(),
		checkTCPReadCase(),
		checkTCPUpdateCase(),
		checkTCPCreateAPIFailureCase(),
		checkTCPReadErrorCase(),
		checkTCPDeleteCase(),
		checkTCPDeleteAPIFailureCase(),
		checkTCPReadNotFoundCase(),
		checkTCPDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
