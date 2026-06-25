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

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const (
	testL7PolicyID = "test-l7policy-id"
	testL7RuleID   = "test-l7rule-id"
	testListenerID = "test-listener-id"
)

func sampleL7Policy(id string) *edgecloud.L7Policy {
	return &edgecloud.L7Policy{
		ID:                 id,
		Name:               "test-l7policy",
		ListenerID:         testListenerID,
		Action:             edgecloud.L7PolicyActionRedirectPrefix,
		ProjectID:          testProjectID,
		RegionID:           testRegionID,
		Region:             "test-region",
		RedirectPrefix:     strPtr("https://redirect.example.com/"),
		RedirectHTTPCode:   intPtr(302),
		Position:           1,
		Rules:              []edgecloud.L7Rule{},
		Tags:               []string{},
		OperatingStatus:    "ONLINE",
		ProvisioningStatus: "ACTIVE",
		CreatedAt:          "2024-01-01T00:00:00Z",
		UpdatedAt:          "2024-01-01T00:00:00Z",
	}
}

func sampleL7Rule(id string) *edgecloud.L7Rule {
	return &edgecloud.L7Rule{
		ID:                 id,
		ProjectID:          testProjectID,
		RegionID:           testRegionID,
		Region:             "test-region",
		Type:               edgecloud.L7RuleTypePath,
		CompareType:        edgecloud.L7RuleCompareTypeRegex,
		Value:              "/images*",
		Key:                "",
		Invert:             false,
		Tags:               []string{},
		OperatingStatus:    "ONLINE",
		ProvisioningStatus: "ACTIVE",
	}
}

func sampleL7RuleUpdated(id string) *edgecloud.L7Rule {
	r := sampleL7Rule(id)
	r.Value = "/new-path*"
	return r
}

func l7ruleCreateCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.L7Rules.On("Create", mock.Anything, l7policyID,
		mock.MatchedBy(func(req *edgecloud.L7RuleCreateRequest) bool {
			return req.Type == edgecloud.L7RuleTypePath &&
				req.CompareType == edgecloud.L7RuleCompareTypeRegex &&
				req.Value == "/images*"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-rule-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-rule-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"l7rules": []interface{}{l7ruleID},
			},
		}, nil, nil)

	mc.L7Rules.On("Get", mock.Anything, l7policyID, l7ruleID).
		Return(sampleL7Rule(l7ruleID), nil, nil)

	mc.L7Policies.On("Get", mock.Anything, l7policyID).
		Return(sampleL7Policy(l7policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/images*",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, l7ruleID)
			support.RequireStateAttrs(t, state, map[string]string{
				"type":          string(edgecloud.L7RuleTypePath),
				"compare_type":  string(edgecloud.L7RuleCompareTypeRegex),
				"value":         "/images*",
				"invert":        "false",
				"l7policy_id":   l7policyID,
				"listener_id":   testListenerID,
				"operating_status":    "ONLINE",
				"provisioning_status": "ACTIVE",
			})
		},
	}
}

func l7ruleReadCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Rules.On("Get", mock.Anything, l7policyID, l7ruleID).
		Return(sampleL7Rule(l7ruleID), nil, nil)

	mc.L7Policies.On("Get", mock.Anything, l7policyID).
		Return(sampleL7Policy(l7policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing l7rule",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: l7ruleID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id": l7policyID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, l7ruleID)
			support.RequireStateAttrs(t, state, map[string]string{
				"type":          string(edgecloud.L7RuleTypePath),
				"compare_type":  string(edgecloud.L7RuleCompareTypeRegex),
				"value":         "/images*",
				"invert":        "false",
				"l7policy_id":   l7policyID,
				"listener_id":   testListenerID,
				"operating_status":    "ONLINE",
				"provisioning_status": "ACTIVE",
			})
		},
	}
}

func l7ruleReadNotFoundCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Rules.On("Get", mock.Anything, l7policyID, l7ruleID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: l7ruleID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id": l7policyID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
		},
	}
}

func l7ruleUpdateValueCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.L7Rules.On("Update", mock.Anything, l7policyID, l7ruleID,
		mock.MatchedBy(func(req *edgecloud.L7RuleUpdateRequest) bool {
			return req.Value != nil && *req.Value == "/new-path*"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-rule-upd"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-rule-upd").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.L7Rules.On("Get", mock.Anything, l7policyID, l7ruleID).
		Return(sampleL7RuleUpdated(l7ruleID), nil, nil)

	mc.L7Policies.On("Get", mock.Anything, l7policyID).
		Return(sampleL7Policy(l7policyID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update value",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: l7ruleID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/images*",
				"invert":       false,
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/new-path*",
				"invert":       false,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, l7ruleID)
			support.RequireStateAttrs(t, state, map[string]string{
				"value": "/new-path*",
			})
		},
	}
}

func l7ruleDeleteCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Rules.On("Delete", mock.Anything, l7policyID, l7ruleID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-rule-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-rule-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete l7rule",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: l7ruleID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/images*",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateID(t, state, l7ruleID)
		},
	}
}

func l7ruleCreateAPIFailureCase(l7policyID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Rules.On("Create", mock.Anything, l7policyID,
		mock.MatchedBy(func(req *edgecloud.L7RuleCreateRequest) bool {
			return req.Value == "/fail-rule"
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/fail-rule",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func l7ruleDeleteTaskErrorCase(l7policyID, l7ruleID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.L7Rules.On("Delete", mock.Anything, l7policyID, l7ruleID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-rule-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-rule-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: l7ruleID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"l7policy_id":  l7policyID,
				"type":         string(edgecloud.L7RuleTypePath),
				"compare_type": string(edgecloud.L7RuleCompareTypeRegex),
				"value":        "/images*",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, l7ruleID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationL7Rule_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_lb_l7rule"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		l7ruleCreateCase(testL7PolicyID, testL7RuleID),
		l7ruleReadCase(testL7PolicyID, testL7RuleID),
		l7ruleReadNotFoundCase(testL7PolicyID, testL7RuleID),
		l7ruleUpdateValueCase(testL7PolicyID, testL7RuleID),
		l7ruleDeleteCase(testL7PolicyID, testL7RuleID),
		l7ruleCreateAPIFailureCase(testL7PolicyID),
		l7ruleDeleteTaskErrorCase(testL7PolicyID, testL7RuleID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
