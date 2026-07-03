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

const testLBPoolID = "test-lb-pool-id"

func samplePool(id string) *edgecloud.Pool {
	return &edgecloud.Pool{
		ID:                    id,
		Name:                  "test-pool",
		LoadbalancerAlgorithm: edgecloud.LoadbalancerAlgorithmRoundRobin,
		Protocol:              edgecloud.LBPoolProtocolHTTP,
		Loadbalancers:         []edgecloud.ID{{ID: testLoadBalancerID}},
		Listeners:             []edgecloud.ID{},
		Members:               []edgecloud.PoolMember{},
		ProvisioningStatus:    edgecloud.ProvisioningStatusActive,
		OperatingStatus:       edgecloud.OperatingStatusOnline,
	}
}

func samplePoolUpdated(id string) *edgecloud.Pool {
	p := samplePool(id)
	p.LoadbalancerAlgorithm = edgecloud.LoadbalancerAlgorithmLeastConnections
	return p
}

func poolCreateCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("PoolCreate", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.PoolCreateRequest) bool {
			return req.Name == "test-pool" &&
				req.Protocol == edgecloud.LBPoolProtocolHTTP &&
				req.LoadbalancerAlgorithm == edgecloud.LoadbalancerAlgorithmRoundRobin &&
				req.LoadbalancerID == testLoadBalancerID
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-pool-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pool-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"pools": []interface{}{poolID},
			},
		}, nil, nil)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePool(poolID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "test-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmRoundRobin),
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, poolID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":            "test-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmRoundRobin),
				"loadbalancer_id": testLoadBalancerID,
			})
		},
	}
}

func poolReadCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePool(poolID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing pool",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, poolID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":            "test-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmRoundRobin),
				"loadbalancer_id": testLoadBalancerID,
			})
		},
	}
}

func poolReadNotFoundCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
		},
	}
}

func poolUpdateAlgorithmCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("PoolUpdate", mock.Anything, poolID,
		mock.MatchedBy(func(req *edgecloud.PoolUpdateRequest) bool {
			return req.LoadbalancerAlgorithm == edgecloud.LoadbalancerAlgorithmLeastConnections
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-pool-upd"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pool-upd").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePoolUpdated(poolID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update algorithm",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "test-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmRoundRobin),
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "test-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmLeastConnections),
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, poolID)
			support.RequireStateAttrs(t, state, map[string]string{
				"lb_algorithm": string(edgecloud.LoadbalancerAlgorithmLeastConnections),
			})
		},
	}
}

func poolDeleteCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolDelete", mock.Anything, poolID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-pool-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pool-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete pool",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
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

func poolCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolCreate", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.PoolCreateRequest) bool {
			return req.Name == "fail-pool"
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name":            "fail-pool",
				"protocol":        string(edgecloud.LBPoolProtocolHTTP),
				"lb_algorithm":    string(edgecloud.LoadbalancerAlgorithmRoundRobin),
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func poolDeleteTaskErrorCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolDelete", mock.Anything, poolID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-pool-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-pool-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
			Error: strPtr("internal error"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, poolID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func poolDeleteNotFoundCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolDelete", mock.Anything, poolID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete non-existent (404)",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: poolID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"loadbalancer_id": testLoadBalancerID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after deleting non-existent resource")
		},
	}
}

func TestIntegrationLBPool_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_lbpool"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		poolCreateCase(testLBPoolID),
		poolReadCase(testLBPoolID),
		poolReadNotFoundCase(testLBPoolID),
		poolUpdateAlgorithmCase(testLBPoolID),
		poolDeleteCase(testLBPoolID),
		poolCreateAPIFailureCase(),
		poolDeleteTaskErrorCase(testLBPoolID),
		poolDeleteNotFoundCase(testLBPoolID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
