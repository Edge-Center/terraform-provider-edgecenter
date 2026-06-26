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

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const (
	testInstanceV2ID  = "inst-v2-id"
	testInstV2VolID   = "vol-v2-id"
	testInstV2DefSGID = "sg-default-v2"
)

func sampleInstanceV2(id, name, flavorID, status, vmState string) *edgecloud.Instance {
	return &edgecloud.Instance{
		ID:     id,
		Name:   name,
		Flavor: &edgecloud.Flavor{FlavorID: flavorID, FlavorName: "g1-standard-2-4", RAM: 4096, VCPUS: 2},
		Status: status,
		VMState: vmState,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
		Metadata:  edgecloud.Metadata{},
	}
}

func sampleInstV2Volume(id, name string, size int, bootable bool) edgecloud.Volume {
	return edgecloud.Volume{
		ID:       id,
		Name:     name,
		Size:     size,
		VolumeType: edgecloud.VolumeTypeStandard,
		Bootable: bootable,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
	}
}

func sampleInstV2ExtIface(portID, networkID string) edgecloud.InstancePortInterface {
	return edgecloud.InstancePortInterface{
		PortID:     portID,
		NetworkID:  networkID,
		MacAddress: "00:00:00:00:00:01",
		NetworkDetails: edgecloud.NetworkSubnetwork{
			Name:     "external-net",
			External: true,
		},
		PortSecurityEnabled: true,
		IPAssignments: []edgecloud.PortIP{
			{SubnetID: "subnet-ext"},
		},
	}
}

func instanceV2CreateCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	defaultSG := edgecloud.SecurityGroup{ID: testInstV2DefSGID, Name: "default"}
	mc.SecurityGroups.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.SecurityGroup{defaultSG}, nil, nil).Once()

	mc.Volumes.On("Get", mock.Anything, testInstV2VolID).
		Return(&edgecloud.Volume{
			ID: testInstV2VolID, Name: "boot-volume", Size: 10, Bootable: true,
		}, nil, nil).Once()

	mc.Instances.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.InstanceCreateRequest) bool {
			return req.Flavor == "g1-standard-2-4" &&
				len(req.Names) == 1 && req.Names[0] == "test-instance" &&
				len(req.Volumes) == 1 && req.Volumes[0].VolumeID == testInstV2VolID &&
				len(req.Interfaces) == 1 && req.Interfaces[0].Type == edgecloud.InterfaceTypeExternal
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"instances": []interface{}{instID},
			},
		}, nil, nil)

	mc.Instances.On("Get", mock.Anything, instID).
		Return(sampleInstanceV2(instID, "test-instance", "g1-standard-2-4", "ACTIVE", "active"), nil, nil)

	mc.Volumes.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Volume{
			sampleInstV2Volume(testInstV2VolID, "boot-volume", 10, true),
		}, nil, nil)

	mc.Instances.On("InterfaceList", mock.Anything, instID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstV2ExtIface("port-1", "net-ext"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-instance"),
			map[string]interface{}{
				"flavor_id": "g1-standard-2-4",
				"boot_volumes": []interface{}{
					map[string]interface{}{
						"volume_id":      testInstV2VolID,
						"boot_index":     0,
						"attachment_tag": "",
					},
				},
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":       "external",
						"is_default": true,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, instID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      "test-instance",
				"flavor_id": "g1-standard-2-4",
				"status":    "ACTIVE",
			})
		},
	}
}

func instanceV2ReadCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("Get", mock.Anything, instID).
		Return(sampleInstanceV2(instID, "test-instance", "g1-standard-2-4", "ACTIVE", "active"), nil, nil)

	mc.Volumes.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Volume{
			sampleInstV2Volume(testInstV2VolID, "boot-volume", 10, true),
		}, nil, nil)

	mc.Instances.On("InterfaceList", mock.Anything, instID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstV2ExtIface("port-1", "net-ext"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing instance",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: instID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-instance"),
			map[string]interface{}{
				"flavor_id": "g1-standard-2-4",
				"boot_volumes": []interface{}{
					map[string]interface{}{
						"volume_id":      testInstV2VolID,
						"boot_index":     0,
						"attachment_tag": "",
					},
				},
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":       "external",
						"is_default": true,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, instID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      "test-instance",
				"flavor_id": "g1-standard-2-4",
				"status":    "ACTIVE",
			})
		},
	}
}

func instanceV2ReadNotFoundCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("Get", mock.Anything, instID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: instID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func instanceV2UpdateNameCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Volumes.On("Get", mock.Anything, testInstV2VolID).
		Return(&edgecloud.Volume{
			ID: testInstV2VolID, Name: "boot-volume", Size: 10, Bootable: true,
		}, nil, nil).Once()

	mc.Instances.On("Rename", mock.Anything, instID,
		mock.MatchedBy(func(n *edgecloud.Name) bool {
			return n.Name == "updated-instance"
		}),
	).Return(sampleInstanceV2(instID, "updated-instance", "g1-standard-2-4", "ACTIVE", "active"), nil, nil)

	mc.Instances.On("Get", mock.Anything, instID).
		Return(sampleInstanceV2(instID, "updated-instance", "g1-standard-2-4", "ACTIVE", "active"), nil, nil)

	mc.Volumes.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Volume{
			sampleInstV2Volume(testInstV2VolID, "boot-volume", 10, true),
		}, nil, nil)

	mc.Instances.On("InterfaceList", mock.Anything, instID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstV2ExtIface("port-1", "net-ext"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: instID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-instance"),
			map[string]interface{}{
				"flavor_id": "g1-standard-2-4",
				"boot_volumes": []interface{}{
					map[string]interface{}{
						"volume_id":      testInstV2VolID,
						"boot_index":     0,
						"attachment_tag": "",
					},
				},
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":       "external",
						"is_default": true,
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("updated-instance"),
			map[string]interface{}{
				"flavor_id": "g1-standard-2-4",
				"boot_volumes": []interface{}{
					map[string]interface{}{
						"volume_id":      testInstV2VolID,
						"boot_index":     0,
						"attachment_tag": "",
					},
				},
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":       "external",
						"is_default": true,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, instID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "updated-instance",
			})
		},
	}
}

func instanceV2DeleteCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("Delete", mock.Anything, instID, mock.Anything).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete instance",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: instID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func instanceV2CreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Volumes.On("Get", mock.Anything, testInstV2VolID).
		Return(&edgecloud.Volume{
			ID: testInstV2VolID, Name: "boot-volume", Size: 10, Bootable: true,
		}, nil, nil).Once()

	mc.SecurityGroups.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.SecurityGroup{{ID: testInstV2DefSGID, Name: "default"}}, nil, nil).Once()

	mc.Instances.On("Create", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-instance"),
			map[string]interface{}{
				"flavor_id": "g1-standard-2-4",
				"boot_volumes": []interface{}{
					map[string]interface{}{
						"volume_id":      testInstV2VolID,
						"boot_index":     0,
						"attachment_tag": "",
					},
				},
				"interfaces": []interface{}{
					map[string]interface{}{
						"type":       "external",
						"is_default": true,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func instanceV2DeleteTaskErrorCase(instID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("Delete", mock.Anything, instID, mock.Anything).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: instID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, instID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationInstanceV2_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_instanceV2"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		instanceV2CreateCase(testInstanceV2ID),
		instanceV2ReadCase(testInstanceV2ID),
		instanceV2ReadNotFoundCase(testInstanceV2ID),
		instanceV2UpdateNameCase(testInstanceV2ID),
		instanceV2DeleteCase(testInstanceV2ID),
		instanceV2CreateAPIFailureCase(),
		instanceV2DeleteTaskErrorCase(testInstanceV2ID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
