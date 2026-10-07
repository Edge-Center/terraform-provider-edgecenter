//go:build integration

package edgecenter_test

import (
	"testing"

	"github.com/stretchr/testify/mock"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testBaremetalID         = "baremetal-id"
	testBaremetalOldImageID = "old-image-id"
	testBaremetalNewImageID = "new-image-id"
)

func baremetalConfig(imageID string) map[string]interface{} {
	return cloud.Merge(
		cloud.WithProjectRegion(testProjectID, testRegionID),
		cloud.WithName("test-baremetal"),
		map[string]interface{}{
			"flavor_id": "bm1-hf-medium",
			"image_id":  imageID,
			"interface": []interface{}{
				map[string]interface{}{
					"type": "external",
				},
			},
		},
	)
}

func baremetalUpdateImageCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Instances.On(
		"BareMetalRebuildInstance",
		mock.Anything,
		testBaremetalID,
		mock.MatchedBy(func(req *edgecloud.BareMetalRebuildRequest) bool {
			return req.ImageID == testBaremetalNewImageID
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-baremetal-rebuild"}}, nil, nil).Once()

	mc.Tasks.On("Get", mock.Anything, "task-baremetal-rebuild").
		Return(&edgecloud.Task{State: edgecloud.TaskStateFinished}, nil, nil).Once()

	mc.Instances.On("Get", mock.Anything, testBaremetalID).
		Return(
			sampleInstanceV2(testBaremetalID, "test-baremetal", "bm1-hf-medium", "ACTIVE", "active"),
			nil,
			nil,
		).Once()

	mc.Instances.On("InterfaceList", mock.Anything, testBaremetalID).
		Return(
			[]edgecloud.InstancePortInterface{
				sampleInstV2ExtIface("port-baremetal", "network-external"),
			},
			nil,
			nil,
		).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:         "update baremetal image",
		Op:           support.OpApply,
		Prepare:      func() *cloudmock.MockedCloud { return mc },
		CurrentID:    testBaremetalID,
		CurrentState: baremetalConfig(testBaremetalOldImageID),
		NewConfig:    baremetalConfig(testBaremetalNewImageID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testBaremetalID)
			support.RequireStateAttrs(t, state, map[string]string{
				"image_id": testBaremetalNewImageID,
			})
		},
	}
}

func TestIntegrationBaremetalUpdateImage(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_baremetal"]

	support.RunResourceCases(
		t,
		resource,
		[]support.ResourceCase[*cloudmock.MockedCloud]{
			baremetalUpdateImageCase(),
		},
		support.DispatchCase[*cloudmock.MockedCloud],
	)
}
