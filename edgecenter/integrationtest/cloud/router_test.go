//go:build unit

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

const testRouterID = "router-id"

func sampleRouter(id, name string) *edgecloud.Router {
	return &edgecloud.Router{
		ID:        id,
		Name:      name,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
		Interfaces: []edgecloud.RouterInterface{},
		Routes:    []edgecloud.HostRoute{},
		ExternalGatewayInfo: edgecloud.ExternalGatewayInfo{},
	}
}

func routerCreateCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Routers.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.RouterCreateRequest) bool {
			return req.Name == "test-router"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"routers": []interface{}{routerID},
			},
		}, nil, nil)

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(sampleRouter(routerID, "test-router"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, routerID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "test-router",
			})
		},
	}
}

func routerAttachSubnetCase(routerID, subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Routers.On("Attach", mock.Anything, routerID,
		mock.MatchedBy(func(req *edgecloud.RouterAttachRequest) bool {
			return req.SubnetID == subnetID
		}),
	).Return((*edgecloud.Router)(nil), nil, nil)

	routerWithIface := sampleRouter(routerID, "test-router")
	routerWithIface.Interfaces = []edgecloud.RouterInterface{
		{
			PortID:    "port-1",
			NetworkID: "net-1",
			MacAddress: "aa:bb:cc:dd:ee:ff",
			IPAssignments: []edgecloud.PortIP{
				{SubnetID: subnetID},
			},
		},
	}

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(routerWithIface, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "attach subnet to router",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
			map[string]interface{}{
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":      "subnet",
						"subnet_id": subnetID,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, routerID)
		},
	}
}

func routerDeleteCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Routers.On("Delete", mock.Anything, routerID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete router",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func routerCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Routers.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.RouterCreateRequest) bool {
			return req.Name == "fail-router"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func routerDeleteTaskErrorCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Routers.On("Delete", mock.Anything, routerID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "task error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, routerID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func routerReadNonExistentCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
			require.NotNil(t, state, "state must not be cleared when read fails")
			require.Equal(t, routerID, state.ID)
		},
	}
}

func routerUpdateNameCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Routers.On("Update", mock.Anything, routerID,
		mock.MatchedBy(func(req *edgecloud.RouterUpdateRequest) bool {
			return req.Name == "updated-router"
		}),
	).Return((*edgecloud.Router)(nil), nil, nil)

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(sampleRouter(routerID, "updated-router"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update router name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("updated-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, routerID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "updated-router",
			})
		},
	}
}

func routerDetachSubnetCase(routerID, subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Routers.On("Detach", mock.Anything, routerID,
		mock.MatchedBy(func(req *edgecloud.RouterDetachRequest) bool {
			return req.SubnetID == subnetID
		}),
	).Return((*edgecloud.Router)(nil), nil, nil)

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(sampleRouter(routerID, "test-router"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "detach subnet from router",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
			map[string]interface{}{
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":      "subnet",
						"subnet_id": subnetID,
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, routerID)
		},
	}
}

func routerUpdateExternalGatewayInfoCase(routerID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Routers.On("Update", mock.Anything, routerID,
		mock.MatchedBy(func(req *edgecloud.RouterUpdateRequest) bool {
			return req.Name == "test-router" && req.ExternalGatewayInfo != nil && req.ExternalGatewayInfo.NetworkID == "ext-net-id"
		}),
	).Return((*edgecloud.Router)(nil), nil, nil)

	routerWithGW := sampleRouter(routerID, "test-router")
	routerWithGW.ExternalGatewayInfo = edgecloud.ExternalGatewayInfo{
		NetworkID: "ext-net-id",
	}

	mc.Routers.On("Get", mock.Anything, routerID).
		Return(routerWithGW, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update external_gateway_info",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: routerID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-router"),
			map[string]interface{}{
				"external_gateway_info": []interface{}{
					map[string]interface{}{
						"type":       "manual",
						"network_id": "ext-net-id",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, routerID)
		},
	}
}

func TestUnitRouter_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_router"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		routerCreateCase(testRouterID),
		routerAttachSubnetCase(testRouterID, "test-subnet-id"),
		routerDetachSubnetCase(testRouterID, "test-subnet-id"),
		routerUpdateExternalGatewayInfoCase(testRouterID),
		routerDeleteCase(testRouterID),
		routerCreateAPIFailureCase(),
		routerDeleteTaskErrorCase(testRouterID),
		routerReadNonExistentCase(testRouterID),
		routerUpdateNameCase(testRouterID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
