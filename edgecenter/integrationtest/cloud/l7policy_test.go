//go:build integration

package edgecenter_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

func sampleL7PolicyUpdated(id string) *edgecloud.L7Policy {
	p := sampleL7Policy(id)
	p.RedirectPrefix = strPtr("https://new-redirect.example.com/")
	return p
}

func sampleListener(id string) *edgecloud.Listener {
	return &edgecloud.Listener{
		ID:       id,
		Protocol: edgecloud.ListenerProtocolTerminatedHTTPS,
	}
}

func l7policyCreateCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, testListenerID).
		Return(sampleListener(testListenerID), nil, nil)

	mc.L7Policies.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.L7PolicyCreateRequest) bool {
			return req.Action == edgecloud.L7PolicyActionRedirectPrefix &&
				req.RedirectPrefix == "https://redirect.example.com/" &&
				req.ListenerID == testListenerID &&
				req.Name == "test-l7policy"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-pol-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pol-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"l7polices": []interface{}{policyID},
			},
		}, nil, nil)

	mc.L7Policies.On("Get", mock.Anything, policyID).
		Return(sampleL7Policy(policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id":      testListenerID,
				"action":           string(edgecloud.L7PolicyActionRedirectPrefix),
				"name":             "test-l7policy",
				"redirect_prefix":  "https://redirect.example.com/",
				"redirect_http_code": 302,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, policyID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                "test-l7policy",
				"action":              string(edgecloud.L7PolicyActionRedirectPrefix),
				"listener_id":         testListenerID,
				"operating_status":    "ONLINE",
				"provisioning_status": "ACTIVE",
			})
		},
	}
}

func l7policyReadCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Policies.On("Get", mock.Anything, policyID).
		Return(sampleL7Policy(policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing l7policy",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: policyID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id": testListenerID,
				"action":      string(edgecloud.L7PolicyActionRedirectPrefix),
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, policyID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                "test-l7policy",
				"action":              string(edgecloud.L7PolicyActionRedirectPrefix),
				"listener_id":         testListenerID,
				"operating_status":    "ONLINE",
				"provisioning_status": "ACTIVE",
			})
		},
	}
}

func l7policyReadNotFoundCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Policies.On("Get", mock.Anything, policyID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: policyID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id": testListenerID,
				"action":      string(edgecloud.L7PolicyActionRedirectPrefix),
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
		},
	}
}

func l7policyUpdateCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, testListenerID).
		Return(sampleListener(testListenerID), nil, nil)

	mc.L7Policies.On("Update", mock.Anything, policyID,
		mock.MatchedBy(func(req *edgecloud.L7PolicyUpdateRequest) bool {
			return req.Action == edgecloud.L7PolicyActionRedirectPrefix &&
				req.RedirectPrefix == "https://new-redirect.example.com/"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-pol-upd"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pol-upd").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.L7Policies.On("Get", mock.Anything, policyID).
		Return(sampleL7PolicyUpdated(policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update redirect_prefix",
		Op:        support.OpApply,
		Skip:      true, // GetRawConfig() returns null in unit tests — SDK limitation
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: policyID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id":     testListenerID,
				"action":          string(edgecloud.L7PolicyActionRedirectPrefix),
				"name":            "test-l7policy",
				"redirect_prefix": "https://redirect.example.com/",
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id":     testListenerID,
				"action":          string(edgecloud.L7PolicyActionRedirectPrefix),
				"name":            "test-l7policy",
				"redirect_prefix": "https://new-redirect.example.com/",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, policyID)
		},
	}
}

func l7policyDeleteCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Policies.On("Delete", mock.Anything, policyID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-pol-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pol-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete l7policy",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: policyID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id": testListenerID,
				"action":      string(edgecloud.L7PolicyActionRedirectPrefix),
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateID(t, state, policyID)
		},
	}
}

func l7policyCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, testListenerID).
		Return(sampleListener(testListenerID), nil, nil)

	mc.L7Policies.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.L7PolicyCreateRequest) bool {
			return req.Name == "fail-policy"
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id":     testListenerID,
				"action":          string(edgecloud.L7PolicyActionRedirectPrefix),
				"name":            "fail-policy",
				"redirect_prefix": "https://fail.example.com/",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func l7policyDeleteTaskErrorCase(policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Policies.On("Delete", mock.Anything, policyID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-pol-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pol-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: policyID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"listener_id": testListenerID,
				"action":      string(edgecloud.L7PolicyActionRedirectPrefix),
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, policyID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationL7Policy_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_lb_l7policy"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		l7policyCreateCase(testL7PolicyID),
		l7policyReadCase(testL7PolicyID),
		l7policyReadNotFoundCase(testL7PolicyID),
		l7policyUpdateCase(testL7PolicyID),
		l7policyDeleteCase(testL7PolicyID),
		l7policyCreateAPIFailureCase(),
		l7policyDeleteTaskErrorCase(testL7PolicyID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
