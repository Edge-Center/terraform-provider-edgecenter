//go:build unit

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/unittest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/unittest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/unittest/support/cloud/mock"
)

const (
	testProjectID = 1
	testRegionID  = 1
	testNetID     = "test-net-id"
)

func sampleNetwork(id, name string) *edgecloud.Network {
	return &edgecloud.Network{
		ID:        id,
		Name:      name,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
		Type:      string(edgecloud.VXLAN),
		MTU:       1500,
		Metadata:  []edgecloud.MetadataDetailed{},
	}
}

func networkCreateCase(netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Networks.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.NetworkCreateRequest) bool {
			return req.Name == "test-net"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"networks": []interface{}{netID},
			},
		}, nil, nil)

	mc.Networks.On("Get", mock.Anything, netID).
		Return(sampleNetwork(netID, "test-net"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, netID)
			support.RequireStateAttrs(t, state, map[string]string{"name": "test-net"})
		},
	}
}

func networkReadCase(netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Networks.On("Get", mock.Anything, netID).
		Return(sampleNetwork(netID, "test-net"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing network",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: netID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, netID)
			support.RequireStateAttrs(t, state, map[string]string{"name": "test-net", "mtu": "1500"})
		},
	}
}

func networkUpdateNameCase(netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Networks.On("UpdateName", mock.Anything, netID,
		mock.MatchedBy(func(n *edgecloud.Name) bool {
			return n.Name == "updated-net"
		}),
	).Return(nil, nil, nil)

	mc.Networks.On("Get", mock.Anything, netID).
		Return(sampleNetwork(netID, "updated-net"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update network name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: netID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-net"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("updated-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, netID)
			support.RequireStateAttrs(t, state, map[string]string{"name": "updated-net"})
		},
	}
}

func networkCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Networks.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.NetworkCreateRequest) bool {
			return req.Name == "fail-net"
		}),
	).Return(nil, nil, fmt.Errorf("api error: network quota exceeded"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "create api error",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "network quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func networkDeleteTaskErrorCase(netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Networks.On("Delete", mock.Anything, netID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: netID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "task with error state")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, netID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func networkDeleteCase(netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Networks.On("Delete", mock.Anything, netID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete network",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: netID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-net"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestUnitNetwork_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_network"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		networkCreateCase(testNetID),
		networkCreateAPIFailureCase(),
		networkReadCase(testNetID),
		networkUpdateNameCase(testNetID),
		networkDeleteCase(testNetID),
		networkDeleteTaskErrorCase(testNetID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
