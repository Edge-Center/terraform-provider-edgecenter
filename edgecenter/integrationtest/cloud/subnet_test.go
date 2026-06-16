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

const testSubnetID = "subnet-id"

func sampleSubnet(id, name, cidr, networkID string) *edgecloud.Subnetwork {
	return &edgecloud.Subnetwork{
		ID:        id,
		Name:      name,
		CIDR:      cidr,
		NetworkID: networkID,
		EnableDHCP: true,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
		Metadata:  []edgecloud.MetadataDetailed{},
	}
}

func subnetCreateCase(subnetID, netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Subnetworks.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SubnetworkCreateRequest) bool {
			return req.Name == "test-subnet" && req.CIDR == "10.0.1.0/24" && req.NetworkID == netID
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"subnets": []interface{}{subnetID},
			},
		}, nil, nil)

	mc.Subnetworks.On("Get", mock.Anything, subnetID).
		Return(sampleSubnet(subnetID, "test-subnet", "10.0.1.0/24", netID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": netID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, subnetID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "test-subnet",
				"cidr":       "10.0.1.0/24",
				"network_id": netID,
			})
		},
	}
}

func subnetReadCase(subnetID, netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Get", mock.Anything, subnetID).
		Return(sampleSubnet(subnetID, "test-subnet", "10.0.1.0/24", netID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing subnet",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": netID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, subnetID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "test-subnet",
				"cidr": "10.0.1.0/24",
			})
		},
	}
}

func subnetUpdateNameCase(subnetID, netID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Subnetworks.On("Update", mock.Anything, subnetID,
		mock.MatchedBy(func(req *edgecloud.SubnetworkUpdateRequest) bool {
			return req.Name == "updated-subnet"
		}),
	).Return((*edgecloud.Subnetwork)(nil), nil, nil)

	mc.Subnetworks.On("Get", mock.Anything, subnetID).
		Return(sampleSubnet(subnetID, "updated-subnet", "10.0.1.0/24", netID), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update subnet name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": netID,
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("updated-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": netID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, subnetID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "updated-subnet",
			})
		},
	}
}

func subnetDeleteCase(subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Delete", mock.Anything, subnetID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete subnet",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": "net-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func subnetCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SubnetworkCreateRequest) bool {
			return req.Name == "fail-subnet"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": "net-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func subnetDeleteTaskErrorCase(subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Delete", mock.Anything, subnetID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "task error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": "net-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, subnetID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func subnetReadNonExistentCase(subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Get", mock.Anything, subnetID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":      "10.0.1.0/24",
				"gatewayIp": "10.0.1.1",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
			require.NotNil(t, state, "state must not be cleared when read fails")
			require.Equal(t, subnetID, state.ID)
		},
	}
}

func subnetDeleteAlreadyDeletedCase(subnetID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Subnetworks.On("Delete", mock.Anything, subnetID).
		Return(&edgecloud.TaskResponse{}, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete already deleted (404)",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: subnetID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-subnet"),
			map[string]interface{}{
				"cidr":       "10.0.1.0/24",
				"network_id": "net-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil for already deleted resource")
		},
	}
}

func TestIntegrationSubnet_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_subnet"]
	netID := "test-net-id"

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		subnetCreateCase(testSubnetID, netID),
		subnetReadCase(testSubnetID, netID),
		subnetUpdateNameCase(testSubnetID, netID),
		subnetDeleteCase(testSubnetID),
		subnetCreateAPIFailureCase(),
		subnetDeleteTaskErrorCase(testSubnetID),
		subnetReadNonExistentCase(testSubnetID),
		subnetDeleteAlreadyDeletedCase(testSubnetID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
