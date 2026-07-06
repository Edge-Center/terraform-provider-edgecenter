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
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkdns"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckDNSID = 101

func baseCheckDNSConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":          "tf-dns",
		"enabled":       true,
		"place":         "country",
		"entities":      []interface{}{1, 2},
		"ip":            "1.1.1.1",
		"resolver":      "8.8.8.8",
		"record_type":   "a",
		"port":          53,
		"interval":      120,
		"check_timeout": 2,
		"retries":       3,
	}
}

func sampleCheckDNS(name, place, recordType string, entities []int) *checkdns.Response {
	return &checkdns.Response{
		Name:         name,
		Enabled:      1,
		Place:        place,
		Entities:     entities,
		IP:           "1.1.1.1",
		Port:         53,
		Resolver:     "8.8.8.8",
		RecordType:   recordType,
		Interval:     120,
		CheckTimeout: 2,
		Retries:      3,
	}
}

func checkDNSCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checkdns.Request) bool {
			return req.Name == "tf-dns" && req.IP == "1.1.1.1" &&
				req.Resolver == "8.8.8.8" && req.RecordType == "a" &&
				req.Enabled == 1 && req.Place == "country"
		}),
	).Return(&checks.CreateResponse{ID: testCheckDNSID}, nil)

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(sampleCheckDNS("tf-dns", "country", "a", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckDNSID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "tf-dns",
				"enabled":     "true",
				"place":       "country",
				"ip":          "1.1.1.1",
				"resolver":    "8.8.8.8",
				"record_type": "a",
				"entities.#":  "2",
				"entities.0":  "1",
				"entities.1":  "2",
			})
		},
	}
}

func checkDNSCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckDNSID}, nil)

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(sampleCheckDNS("tf-dns", "all", "a", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckDNSConfig(), map[string]interface{}{
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

func checkDNSReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(sampleCheckDNS("tf-dns", "country", "a", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckDNSID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "tf-dns",
				"record_type": "a",
				"entities.#":  "2",
			})
		},
	}
}

func checkDNSUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Update", mock.Anything, testCheckDNSID,
		mock.MatchedBy(func(req *checkdns.Request) bool {
			return req.RecordType == "txt"
		}),
	).Return(nil)

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(sampleCheckDNS("tf-dns", "country", "txt", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update record_type",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		NewConfig: edgemon.Merge(baseCheckDNSConfig(), map[string]interface{}{
			"record_type": "txt",
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"record_type": "txt",
			})
		},
	}
}

func checkDNSCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkDNSReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkDNSDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Delete", mock.Anything, testCheckDNSID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkDNSDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Delete", mock.Anything, testCheckDNSID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckDNSID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkDNSReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Get", mock.Anything, testCheckDNSID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkDNSDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckDNS.On("Delete", mock.Anything, testCheckDNSID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckDNSID),
		CurrentState: baseCheckDNSConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckDNS_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_dns")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkDNSCreateCase(),
		checkDNSCreatePlaceAllCase(),
		checkDNSReadCase(),
		checkDNSUpdateCase(),
		checkDNSCreateAPIFailureCase(),
		checkDNSReadErrorCase(),
		checkDNSDeleteCase(),
		checkDNSDeleteAPIFailureCase(),
		checkDNSReadNotFoundCase(),
		checkDNSDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
