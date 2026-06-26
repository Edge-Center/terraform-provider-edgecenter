//go:build integration

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
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const testFloatingIPID = "fip-id"

func sampleFloatingIP(id, address, status, portID string) edgecloud.FloatingIP {
	return edgecloud.FloatingIP{
		ID:                id,
		FloatingIPAddress: address,
		Status:            status,
		PortID:            portID,
		ProjectID:         testProjectID,
		RegionID:          testRegionID,
		Metadata:          []edgecloud.MetadataDetailed{},
	}
}

func floatingIPCreateCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Floatingips.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.FloatingIPCreateRequest) bool {
			return req.PortID == ""
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"floatingips": []interface{}{fipID},
			},
		}, nil, nil)

	mc.Floatingips.On("List", mock.Anything).
		Return([]edgecloud.FloatingIP{
			sampleFloatingIP(fipID, "10.0.0.1", "DOWN", ""),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fipID)
			support.RequireStateAttrs(t, state, map[string]string{
				"floating_ip_address": "10.0.0.1",
				"status":              "DOWN",
			})
		},
	}
}

func floatingIPAssignCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Floatingips.On("Assign", mock.Anything, fipID,
		mock.MatchedBy(func(req *edgecloud.AssignFloatingIPRequest) bool {
			return req.PortID == "port-1"
		}),
	).Return((*edgecloud.FloatingIP)(nil), nil, nil)

	assignedIP := sampleFloatingIP(fipID, "10.0.0.1", "ACTIVE", "port-1")

	mc.Floatingips.On("List", mock.Anything).
		Return([]edgecloud.FloatingIP{assignedIP}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "assign floating IP to port",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"port_id": "port-1",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fipID)
			support.RequireStateAttrs(t, state, map[string]string{
				"port_id": "port-1",
				"status":  "ACTIVE",
			})
		},
	}
}

func floatingIPUnassignCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Floatingips.On("UnAssign", mock.Anything, fipID).
		Return((*edgecloud.FloatingIP)(nil), nil, nil)

	unassignedIP := sampleFloatingIP(fipID, "10.0.0.1", "DOWN", "")

	mc.Floatingips.On("List", mock.Anything).
		Return([]edgecloud.FloatingIP{unassignedIP}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "unassign floating IP",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"port_id": "port-1",
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fipID)
			support.RequireStateAttrs(t, state, map[string]string{
				"port_id": "",
				"status":  "DOWN",
			})
		},
	}
}

func floatingIPUpdateMetadataCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Floatingips.On("MetadataUpdate", mock.Anything, fipID,
		mock.MatchedBy(func(meta *edgecloud.Metadata) bool {
			return (*meta)["key"] == "value"
		}),
	).Return(nil, nil)

	fipWithMeta := sampleFloatingIP(fipID, "10.0.0.1", "DOWN", "")
	fipWithMeta.Metadata = []edgecloud.MetadataDetailed{
		{Key: "key", Value: "value", ReadOnly: false},
	}

	mc.Floatingips.On("List", mock.Anything).
		Return([]edgecloud.FloatingIP{fipWithMeta}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update metadata",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithMetadata(map[string]string{"key": "value"}),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fipID)
		},
	}
}

func floatingIPDeleteCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Floatingips.On("Delete", mock.Anything, fipID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete floating IP",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func floatingIPCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Floatingips.On("Create", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func floatingIPReadNotFoundCase(fipID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Floatingips.On("List", mock.Anything).
		Return([]edgecloud.FloatingIP{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent floating IP",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fipID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func TestIntegrationFloatingIP_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_floatingip"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		floatingIPCreateCase(testFloatingIPID),
		floatingIPAssignCase(testFloatingIPID),
		floatingIPUnassignCase(testFloatingIPID),
		floatingIPUpdateMetadataCase(testFloatingIPID),
		floatingIPDeleteCase(testFloatingIPID),
		floatingIPCreateAPIFailureCase(),
		floatingIPReadNotFoundCase(testFloatingIPID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
