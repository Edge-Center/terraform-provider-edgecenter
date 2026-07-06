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
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checksmtp"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckSMTPID = 104

func baseCheckSMTPConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":             "tf-smtp",
		"enabled":          true,
		"place":            "country",
		"entities":         []interface{}{1, 2},
		"ip":               "1.1.1.1",
		"port":             587,
		"username":         "user",
		"password":         "pass",
		"ignore_ssl_error": true,
		"interval":         120,
		"check_timeout":    2,
		"retries":          3,
	}
}

func sampleCheckSMTP(name, place, username string, entities []int) *checksmtp.Response {
	return &checksmtp.Response{
		Name:           name,
		Enabled:        1,
		Place:          place,
		Entities:       entities,
		IP:             "1.1.1.1",
		Port:           587,
		IgnoreSSLError: 1,
		Username:       username,
		Password:       "pass",
		Interval:       120,
		CheckTimeout:   2,
		Retries:        3,
	}
}

func checkSMTPCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checksmtp.Request) bool {
			return req.Name == "tf-smtp" && req.IP == "1.1.1.1" &&
				req.Port == 587 && req.Username == "user" &&
				req.Enabled == 1 && req.Place == "country"
		}),
	).Return(&checks.CreateResponse{ID: testCheckSMTPID}, nil)

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(sampleCheckSMTP("tf-smtp", "country", "user", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckSMTPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":             "tf-smtp",
				"enabled":          "true",
				"place":            "country",
				"ip":               "1.1.1.1",
				"port":             "587",
				"username":         "user",
				"ignore_ssl_error": "true",
				"entities.#":       "2",
				"entities.0":       "1",
				"entities.1":       "2",
			})
		},
	}
}

func checkSMTPCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckSMTPID}, nil)

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(sampleCheckSMTP("tf-smtp", "all", "user", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckSMTPConfig(), map[string]interface{}{
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

func checkSMTPReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(sampleCheckSMTP("tf-smtp", "country", "user", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckSMTPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-smtp",
				"username":   "user",
				"port":       "587",
				"entities.#": "2",
			})
		},
	}
}

func checkSMTPUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Update", mock.Anything, testCheckSMTPID,
		mock.MatchedBy(func(req *checksmtp.Request) bool {
			return req.Username == "user2"
		}),
	).Return(nil)

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(sampleCheckSMTP("tf-smtp", "country", "user2", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update username",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		NewConfig: edgemon.Merge(baseCheckSMTPConfig(), map[string]interface{}{
			"username": "user2",
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"username": "user2",
			})
		},
	}
}

func checkSMTPCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkSMTPReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkSMTPDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Delete", mock.Anything, testCheckSMTPID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkSMTPDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Delete", mock.Anything, testCheckSMTPID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckSMTPID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkSMTPReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Get", mock.Anything, testCheckSMTPID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkSMTPDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckSMTP.On("Delete", mock.Anything, testCheckSMTPID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckSMTPID),
		CurrentState: baseCheckSMTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckSMTP_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_smtp")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkSMTPCreateCase(),
		checkSMTPCreatePlaceAllCase(),
		checkSMTPReadCase(),
		checkSMTPUpdateCase(),
		checkSMTPCreateAPIFailureCase(),
		checkSMTPReadErrorCase(),
		checkSMTPDeleteCase(),
		checkSMTPDeleteAPIFailureCase(),
		checkSMTPReadNotFoundCase(),
		checkSMTPDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
