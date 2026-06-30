//go:build integration

package edgecenter_test

import (
	"fmt"
	"net"
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
	testLBMemberID = "test-lb-member-id"
	testLBPoolID2  = "test-lb-pool-id"
)

func samplePoolWithMember(poolID, memberID string) *edgecloud.Pool {
	return &edgecloud.Pool{
		ID:                    poolID,
		Name:                  "test-pool",
		LoadbalancerAlgorithm: edgecloud.LoadbalancerAlgorithmRoundRobin,
		Protocol:              edgecloud.LBPoolProtocolHTTP,
		Loadbalancers:         []edgecloud.ID{{ID: testLoadBalancerID}},
		Members: []edgecloud.PoolMember{
			{
				ID:              memberID,
				OperatingStatus: edgecloud.OperatingStatusOnline,
				PoolMemberCreateRequest: edgecloud.PoolMemberCreateRequest{
					Address:      net.ParseIP("10.0.0.10"),
					ProtocolPort: 8080,
					Weight:       100,
					SubnetID:     "test-subnet-id",
					InstanceID:   "",
				},
			},
		},
		ProvisioningStatus: edgecloud.ProvisioningStatusActive,
		OperatingStatus:    edgecloud.OperatingStatusOnline,
	}
}

func samplePoolWithMemberUpdated(poolID, memberID string) *edgecloud.Pool {
	p := samplePoolWithMember(poolID, memberID)
	p.Members[0].Weight = 200
	return p
}

func memberCreateCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("PoolMemberCreate", mock.Anything, poolID,
		mock.MatchedBy(func(req *edgecloud.PoolMemberCreateRequest) bool {
			return req.ProtocolPort == 8080 &&
				req.Weight == 100 &&
				req.SubnetID == "test-subnet-id" &&
				req.Address.Equal(net.ParseIP("10.0.0.10"))
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-member-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-member-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"members": []interface{}{memberID},
			},
		}, nil, nil)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePoolWithMember(poolID, memberID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id":      poolID,
				"address":      "10.0.0.10",
				"protocol_port": 8080,
				"weight":       100,
				"subnet_id":    "test-subnet-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, memberID)
			support.RequireStateAttrs(t, state, map[string]string{
				"address":          "10.0.0.10",
				"protocol_port":    "8080",
				"weight":           "100",
				"subnet_id":        "test-subnet-id",
				"operating_status": string(edgecloud.OperatingStatusOnline),
			})
		},
	}
}

func memberReadCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePoolWithMember(poolID, memberID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing member",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: memberID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id": poolID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, memberID)
			support.RequireStateAttrs(t, state, map[string]string{
				"address":          "10.0.0.10",
				"protocol_port":    "8080",
				"weight":           "100",
				"subnet_id":        "test-subnet-id",
				"operating_status": string(edgecloud.OperatingStatusOnline),
			})
		},
	}
}

func memberReadNotFoundCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	mc.Loadbalancers.On("PoolList", mock.Anything, mock.Anything).
		Return([]edgecloud.Pool{}, &edgecloud.Response{}, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404) -> rebind clears state",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: memberID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id": poolID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func memberUpdateWeightCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePoolWithMember(poolID, memberID), nil, nil).Once()

	mc.Loadbalancers.On("PoolUpdate", mock.Anything, poolID,
		mock.MatchedBy(func(req *edgecloud.PoolUpdateRequest) bool {
			return len(req.Members) == 1 &&
				req.Members[0].ID == memberID &&
				req.Members[0].Weight == 200
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-member-upd"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-member-upd").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.Loadbalancers.On("PoolGet", mock.Anything, poolID).
		Return(samplePoolWithMemberUpdated(poolID, memberID), nil, nil).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update weight",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: memberID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id":       poolID,
				"address":       "10.0.0.10",
				"protocol_port": 8080,
				"weight":        100,
				"subnet_id":     "test-subnet-id",
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id":       poolID,
				"address":       "10.0.0.10",
				"protocol_port": 8080,
				"weight":        200,
				"subnet_id":     "test-subnet-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, memberID)
			support.RequireStateAttrs(t, state, map[string]string{
				"weight": "200",
			})
		},
	}
}

func memberDeleteCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolMemberDelete", mock.Anything, poolID, memberID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-member-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-member-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete member",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: memberID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id": poolID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func memberCreateAPIFailureCase(poolID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolMemberCreate", mock.Anything, poolID,
		mock.MatchedBy(func(req *edgecloud.PoolMemberCreateRequest) bool {
			return req.ProtocolPort == 9090
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id":       poolID,
				"address":       "10.0.0.20",
				"protocol_port": 9090,
				"weight":        50,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func memberDeleteTaskErrorCase(poolID, memberID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("PoolMemberDelete", mock.Anything, poolID, memberID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-member-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-member-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
			Error: strPtr("internal error"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: memberID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"pool_id": poolID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, memberID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationLBMember_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_lbmember"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		memberCreateCase(testLBPoolID2, testLBMemberID),
		memberReadCase(testLBPoolID2, testLBMemberID),
		memberReadNotFoundCase(testLBPoolID2, testLBMemberID),
		memberUpdateWeightCase(testLBPoolID2, testLBMemberID),
		memberDeleteCase(testLBPoolID2, testLBMemberID),
		memberCreateAPIFailureCase(testLBPoolID2),
		memberDeleteTaskErrorCase(testLBPoolID2, testLBMemberID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
