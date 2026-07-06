//go:build integration

package edgemon_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenteredgemon-go/statuspage"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testStatusPageID = 107

func baseStatusPageConfig(slug string) map[string]interface{} {
	return map[string]interface{}{
		"name": "tf-page",
		"slug": slug,
		"checks": []interface{}{
			map[string]interface{}{"check_id": 10},
		},
	}
}

func sampleStatusPage(slug string) *statuspage.Response {
	return &statuspage.Response{
		ID: testStatusPageID,
		Base: statuspage.Base{
			Name: "tf-page",
			Slug: slug,
		},
		Checks: []statuspage.Checks{{CheckID: 10}},
	}
}

func statusPageCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Create", mock.Anything,
		mock.MatchedBy(func(req *statuspage.Request) bool {
			return req.Name == "tf-page" && req.Slug == "tf-slug"
		}),
	).Return(&statuspage.CreateResponse{ID: testStatusPageID}, nil)

	mc.StatusPage.On("Get", mock.Anything, testStatusPageID).
		Return(sampleStatusPage("tf-slug"), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testStatusPageID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":              "tf-page",
				"slug":              "tf-slug",
				"checks.#":          "1",
				"checks.0.check_id": "10",
			})
		},
	}
}

func statusPageReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Get", mock.Anything, testStatusPageID).
		Return(sampleStatusPage("tf-slug"), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing status page",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testStatusPageID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":              "tf-page",
				"slug":              "tf-slug",
				"checks.#":          "1",
				"checks.0.check_id": "10",
			})
		},
	}
}

func statusPageUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Update", mock.Anything, testStatusPageID,
		mock.MatchedBy(func(req *statuspage.Request) bool {
			return req.Slug == "tf-slug-2"
		}),
	).Return(nil)

	mc.StatusPage.On("Get", mock.Anything, testStatusPageID).
		Return(sampleStatusPage("tf-slug-2"), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update slug",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		NewConfig:    baseStatusPageConfig("tf-slug-2"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"slug": "tf-slug-2",
			})
		},
	}
}

func statusPageCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: invalid slug"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid slug")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func statusPageDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Delete", mock.Anything, testStatusPageID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete status page",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func statusPageDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Delete", mock.Anything, testStatusPageID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testStatusPageID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func statusPageDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Delete", mock.Anything, testStatusPageID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func statusPageReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.StatusPage.On("Get", mock.Anything, testStatusPageID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testStatusPageID),
		CurrentState: baseStatusPageConfig("tf-slug"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func TestIntegrationStatusPage_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_status_page")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		statusPageCreateCase(),
		statusPageReadCase(),
		statusPageUpdateCase(),
		statusPageCreateAPIFailureCase(),
		statusPageDeleteCase(),
		statusPageDeleteAPIFailureCase(),
		statusPageDeleteNotFoundCase(),
		statusPageReadNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
