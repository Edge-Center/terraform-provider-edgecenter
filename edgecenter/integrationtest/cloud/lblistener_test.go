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

const testLBListenerID = "test-listener-id"

func sampleLBListener(id string) *edgecloud.Listener {
	return &edgecloud.Listener{
		ID:                 id,
		Name:               "test-listener",
		LoadbalancerID:     testLoadBalancerID,
		Protocol:           edgecloud.ListenerProtocolHTTP,
		ProtocolPort:       80,
		PoolCount:          0,
		OperatingStatus:    edgecloud.OperatingStatusOnline,
		ProvisioningStatus: edgecloud.ProvisioningStatusActive,
		AllowedCIDRs:       []string{},
		SNISecretID:        []string{},
		InsertHeaders:      map[string]string{},
	}
}

func sampleLBListenerUpdated(id string) *edgecloud.Listener {
	l := sampleLBListener(id)
	l.Name = "new-listener-name"
	return l
}

func listenerCreateCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("ListenerCreate", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ListenerCreateRequest) bool {
			return req.Name == "test-listener" &&
				req.Protocol == edgecloud.ListenerProtocolHTTP &&
				req.ProtocolPort == 80 &&
				req.LoadbalancerID == testLoadBalancerID
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-ls-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-ls-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"listeners": []interface{}{listenerID},
			},
		}, nil, nil)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, listenerID).
		Return(sampleLBListener(listenerID), nil, nil)

	mc.L7Policies.On("List", mock.Anything).
		Return([]edgecloud.L7Policy{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":             "test-listener",
				"loadbalancer_id":  testLoadBalancerID,
				"protocol":         string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":    80,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, listenerID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                "test-listener",
				"protocol":            string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":       "80",
				"loadbalancer_id":     testLoadBalancerID,
				"pool_count":          "0",
				"operating_status":    string(edgecloud.OperatingStatusOnline),
				"provisioning_status": string(edgecloud.ProvisioningStatusActive),
			})
		},
	}
}

func listenerReadCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, listenerID).
		Return(sampleLBListener(listenerID), nil, nil)

	mc.L7Policies.On("List", mock.Anything).
		Return([]edgecloud.L7Policy{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing listener",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: listenerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, listenerID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                "test-listener",
				"protocol":            string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":       "80",
				"loadbalancer_id":     testLoadBalancerID,
				"pool_count":          "0",
				"operating_status":    string(edgecloud.OperatingStatusOnline),
				"provisioning_status": string(edgecloud.ProvisioningStatusActive),
			})
		},
	}
}

func listenerReadNotFoundCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, listenerID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	mc.Loadbalancers.On("ListenerList", mock.Anything, mock.Anything).
		Return([]edgecloud.Listener{}, &edgecloud.Response{}, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404) -> rebind clears state",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: listenerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func listenerUpdateNameCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("ListenerUpdate", mock.Anything, listenerID,
		mock.MatchedBy(func(req *edgecloud.ListenerUpdateRequest) bool {
			return req.Name == "new-listener-name"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-ls-upd"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-ls-upd").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.Loadbalancers.On("ListenerGet", mock.Anything, listenerID).
		Return(sampleLBListenerUpdated(listenerID), nil, nil)

	mc.L7Policies.On("List", mock.Anything).
		Return([]edgecloud.L7Policy{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: listenerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "test-listener",
				"loadbalancer_id": testLoadBalancerID,
				"protocol":        string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":   80,
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "new-listener-name",
				"loadbalancer_id": testLoadBalancerID,
				"protocol":        string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":   80,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, listenerID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "new-listener-name",
			})
		},
	}
}

func listenerDeleteCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerDelete", mock.Anything, listenerID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-ls-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-ls-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete listener",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: listenerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func listenerCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerCreate", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ListenerCreateRequest) bool {
			return req.Name == "fail-listener"
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "fail-listener",
				"loadbalancer_id": testLoadBalancerID,
				"protocol":        string(edgecloud.ListenerProtocolHTTP),
				"protocol_port":   80,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func listenerDeleteTaskErrorCase(listenerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("ListenerDelete", mock.Anything, listenerID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-ls-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-ls-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
			Error: strPtr("internal error"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: listenerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, listenerID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationLBListener_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_lblistener"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		listenerCreateCase(testLBListenerID),
		listenerReadCase(testLBListenerID),
		listenerReadNotFoundCase(testLBListenerID),
		listenerUpdateNameCase(testLBListenerID),
		listenerDeleteCase(testLBListenerID),
		listenerCreateAPIFailureCase(),
		listenerDeleteTaskErrorCase(testLBListenerID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
