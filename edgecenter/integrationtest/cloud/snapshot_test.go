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

const testSnapshotID = "snap-id"

func sampleSnapshot(id, name string) *edgecloud.Snapshot {
	return &edgecloud.Snapshot{
		ID:          id,
		Name:        name,
		Status:      "available",
		Size:        10,
		VolumeID:    "vol-id",
		Description: "test snapshot",
		ProjectID:   testProjectID,
		RegionID:    testRegionID,
	}
}

func snapshotCreateCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Snapshots.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SnapshotCreateRequest) bool {
			return req.Name == "test-snapshot" && req.VolumeID == "vol-id"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-snap-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-snap-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"snapshots": []interface{}{snapID},
			},
		}, nil, nil)

	mc.Snapshots.On("Get", mock.Anything, snapID).
		Return(sampleSnapshot(snapID, "test-snapshot"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id":   "vol-id",
				"description": "test snapshot",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, snapID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "test-snapshot",
				"status":      "available",
				"size":        "10",
				"volume_id":   "vol-id",
				"description": "test snapshot",
			})
		},
	}
}

func snapshotReadCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Get", mock.Anything, snapID).
		Return(sampleSnapshot(snapID, "test-snapshot"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing snapshot",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, snapID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   "test-snapshot",
				"status": "available",
				"size":   "10",
			})
		},
	}
}

func snapshotDeleteCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Delete", mock.Anything, snapID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete snapshot",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func snapshotCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SnapshotCreateRequest) bool {
			return req.Name == "fail-snapshot"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func snapshotReadNonExistentCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Get", mock.Anything, snapID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state should not be nil even on 404")
		},
	}
}

func snapshotDeleteOnDeletedCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Delete", mock.Anything, snapID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete on already-deleted (404)",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource already deleted")
		},
	}
}

func snapshotDeleteTaskErrorCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Snapshots.On("Delete", mock.Anything, snapID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "task error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, snapID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func snapshotUpdateMetadataCase(snapID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	updated := sampleSnapshot(snapID, "test-snapshot")
	updated.Metadata = edgecloud.Metadata{
		"key": "value",
	}

	mc.Snapshots.On("MetadataUpdate", mock.Anything, snapID,
		mock.MatchedBy(func(req *edgecloud.MetadataCreateRequest) bool {
			return req.Metadata["key"] == "value"
		}),
	).Return(sampleSnapshot(snapID, "test-snapshot"), nil, nil)

	mc.Snapshots.On("Get", mock.Anything, snapID).
		Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update metadata",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: snapID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-snapshot"),
			map[string]interface{}{
				"volume_id": "vol-id",
				"metadata": map[string]interface{}{
					"key": "value",
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, snapID)
			support.RequireStateAttrs(t, state, map[string]string{
				"metadata.%":     "1",
				"metadata.key":   "value",
			})
		},
	}
}

func TestIntegrationSnapshot_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_snapshot"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		snapshotCreateCase(testSnapshotID),
		snapshotReadCase(testSnapshotID),
		snapshotUpdateMetadataCase(testSnapshotID),
		snapshotDeleteCase(testSnapshotID),
		snapshotCreateAPIFailureCase(),
		snapshotReadNonExistentCase(testSnapshotID),
		snapshotDeleteTaskErrorCase(testSnapshotID),
		snapshotDeleteOnDeletedCase(testSnapshotID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
