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
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkhttp"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckHTTPID = 77

func baseCheckHTTPConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":                  "tf-http",
		"enabled":               true,
		"place":                 "country",
		"entities":              []interface{}{1, 2},
		"url":                   "https://example.com",
		"method":                "get",
		"accepted_status_codes": []interface{}{200, 201},
		"ignore_ssl_error":      true,
		"interval":              120,
		"check_timeout":         2,
		"retries":               3,
		"redirects":             3,
	}
}

func sampleCheckHTTP(name, place, method string, entities []int) *checkhttp.Response {
	return &checkhttp.Response{
		Name:                name,
		Enabled:             1,
		Place:               place,
		Entities:            entities,
		URL:                 "https://example.com",
		Method:              method,
		AcceptedStatusCodes: []int{200, 201},
		IgnoreSSLError:      1,
		Interval:            120,
		CheckTimeout:        2,
		Retries:             3,
		Redirects:           3,
	}
}

func checkHTTPCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checkhttp.Request) bool {
			return req.Name == "tf-http" && req.URL == "https://example.com" &&
				req.Method == "get" && req.Enabled == 1 && req.Place == "country"
		}),
	).Return(&checks.CreateResponse{ID: testCheckHTTPID}, nil)

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(sampleCheckHTTP("tf-http", "country", "get", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckHTTPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                    "tf-http",
				"enabled":                 "true",
				"place":                   "country",
				"method":                  "get",
				"url":                     "https://example.com",
				"ignore_ssl_error":        "true",
				"entities.#":              "2",
				"entities.0":              "1",
				"entities.1":              "2",
				"accepted_status_codes.#": "2",
				"accepted_status_codes.0": "200",
			})
		},
	}
}

func checkHTTPCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckHTTPID}, nil)

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(sampleCheckHTTP("tf-http", "all", "get", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckHTTPConfig(), map[string]interface{}{
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

func checkHTTPReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(sampleCheckHTTP("tf-http", "country", "get", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckHTTPID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-http",
				"method":     "get",
				"entities.#": "2",
			})
		},
	}
}

func checkHTTPUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Update", mock.Anything, testCheckHTTPID,
		mock.MatchedBy(func(req *checkhttp.Request) bool {
			return req.Method == "post"
		}),
	).Return(nil)

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(sampleCheckHTTP("tf-http", "country", "post", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update method",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		NewConfig: edgemon.Merge(baseCheckHTTPConfig(), map[string]interface{}{
			"method": "post",
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"method": "post",
			})
		},
	}
}

func checkHTTPCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkHTTPReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkHTTPDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Delete", mock.Anything, testCheckHTTPID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkHTTPDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Delete", mock.Anything, testCheckHTTPID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckHTTPID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkHTTPReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Get", mock.Anything, testCheckHTTPID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkHTTPDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckHTTP.On("Delete", mock.Anything, testCheckHTTPID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckHTTPID),
		CurrentState: baseCheckHTTPConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckHTTP_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_http")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkHTTPCreateCase(),
		checkHTTPCreatePlaceAllCase(),
		checkHTTPReadCase(),
		checkHTTPUpdateCase(),
		checkHTTPCreateAPIFailureCase(),
		checkHTTPReadErrorCase(),
		checkHTTPDeleteCase(),
		checkHTTPDeleteAPIFailureCase(),
		checkHTTPReadNotFoundCase(),
		checkHTTPDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
