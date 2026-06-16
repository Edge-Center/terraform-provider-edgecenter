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

const testVolumeID = "vol-id"

func sampleVolume(id, name string, size int) *edgecloud.Volume {
	return &edgecloud.Volume{
		ID:          id,
		Name:        name,
		Size:        size,
		VolumeType:  edgecloud.VolumeTypeStandard,
		ProjectID:   testProjectID,
		RegionID:    testRegionID,
		Attachments: []edgecloud.Attachment{},
	}
}

func volumeCreateCase(volID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Volumes.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.VolumeCreateRequest) bool {
			return req.Name == "test-volume" && req.Size == 10 && req.TypeName == edgecloud.VolumeTypeStandard
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-vol-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-vol-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"volumes": []interface{}{volID},
			},
		}, nil, nil)

	mc.Volumes.On("Get", mock.Anything, volID).
		Return(sampleVolume(volID, "test-volume", 10), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-volume"),
			cloud.WithSize(10),
			cloud.WithTypeName("standard"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, volID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      "test-volume",
				"size":      "10",
				"type_name": "standard",
			})
		},
	}
}

func volumeDeleteCase(volID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Volumes.On("Get", mock.Anything, volID).
		Return(sampleVolume(volID, "test-volume", 10), nil, nil).Once()

	mc.Volumes.On("Delete", mock.Anything, volID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	mc.Volumes.On("Get", mock.Anything, volID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found")).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete volume",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: volID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-volume"),
			cloud.WithSize(10),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestUnitVolume_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_volume"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		volumeCreateCase(testVolumeID),
		volumeDeleteCase(testVolumeID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
