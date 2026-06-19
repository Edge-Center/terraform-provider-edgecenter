//go:build unit

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

const testRFIPID = "rfip-id"

func sampleReservedFixedIP(id string) *edgecloud.ReservedFixedIP {
	return &edgecloud.ReservedFixedIP{
		PortID:         id,
		FixedIPAddress: net.ParseIP("10.0.0.1"),
		Name:           "test-rfip",
		Status:         "ACTIVE",
		IsVIP:          false,
		IsExternal:     true,
		ProjectID:      testProjectID,
		RegionID:       testRegionID,
		SubnetID:       "subnet-id",
		NetworkID:      "network-id",
		Reservation: edgecloud.Reservation{
			Status:       "active",
			ResourceType: "network",
			ResourceID:   "res-id",
		},
	}
}

func reservedFixedIPCreateExternalCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.ReservedFixedIP.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ReservedFixedIPCreateRequest) bool {
			return req.Type == edgecloud.ReservedFixedIPTypeExternal
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-rfip-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-rfip-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"ports": []interface{}{ipID},
			},
		}, nil, nil)

	mc.ReservedFixedIP.On("Get", mock.Anything, ipID).
		Return(sampleReservedFixedIP(ipID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create external",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, ipID)
			support.RequireStateAttrs(t, state, map[string]string{
				"type":             "external",
				"status":           "ACTIVE",
				"port_id":          ipID,
				"fixed_ip_address": "10.0.0.1",
			})
		},
	}
}

func reservedFixedIPReadCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Get", mock.Anything, ipID).
		Return(sampleReservedFixedIP(ipID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing reserved fixed IP",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: ipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, ipID)
			support.RequireStateAttrs(t, state, map[string]string{
				"status":           "ACTIVE",
				"fixed_ip_address": "10.0.0.1",
			})
		},
	}
}

func reservedFixedIPDeleteCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Delete", mock.Anything, ipID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete reserved fixed IP",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: ipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func reservedFixedIPCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ReservedFixedIPCreateRequest) bool {
			return req.Type == edgecloud.ReservedFixedIPTypeExternal
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func reservedFixedIPReadNonExistentCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Get", mock.Anything, ipID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: ipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func reservedFixedIPDeleteTaskErrorCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Delete", mock.Anything, ipID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "task error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: ipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, ipID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func reservedFixedIPDeleteOnDeletedCase(ipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ReservedFixedIP.On("Delete", mock.Anything, ipID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete on already-deleted (404)",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: ipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"type": "external",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource already deleted")
		},
	}
}

func TestUnitReservedFixedIP_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_reservedfixedip"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		reservedFixedIPCreateExternalCase(testRFIPID),
		reservedFixedIPReadCase(testRFIPID),
		reservedFixedIPDeleteCase(testRFIPID),
		reservedFixedIPCreateAPIFailureCase(),
		reservedFixedIPReadNonExistentCase(testRFIPID),
		reservedFixedIPDeleteTaskErrorCase(testRFIPID),
		reservedFixedIPDeleteOnDeletedCase(testRFIPID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
