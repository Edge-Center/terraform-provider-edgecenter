//go:build integration

package edgecenter_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

const testBaremetalAppTemplateID = "app-template-id"

func baremetalConfig(imageID string) map[string]interface{} {
	return baremetalSourceConfig(map[string]interface{}{"image_id": imageID})
}

func baremetalSourceConfig(source map[string]interface{}) map[string]interface{} {
	return cloud.Merge(
		cloud.WithProjectRegion(testProjectID, testRegionID),
		cloud.WithName("test-baremetal"),
		map[string]interface{}{
			"flavor_id": "bm1-hf-medium",
			"interface": []interface{}{
				map[string]interface{}{
					"type": "external",
				},
			},
		},
		source,
	)
}

func baremetalInstance(installedImageID, taskID string) *edgecloud.Instance {
	instance := sampleInstanceV2(testBaremetalID, "test-baremetal", "bm1-hf-medium", "ACTIVE", "active")
	if installedImageID != "" {
		instance.Metadata = edgecloud.Metadata{"image_id": installedImageID}
	}
	instance.TaskID = taskID

	return instance
}

func expectBaremetalRead(mc *cloudmock.MockedCloud, instance *edgecloud.Instance, gets int) {
	mc.Instances.On("Get", mock.Anything, testBaremetalID).Return(instance, nil, nil).Times(gets)

	mc.Instances.On("InterfaceList", mock.Anything, testBaremetalID).
		Return(
			[]edgecloud.InstancePortInterface{
				sampleInstV2ExtIface("port-baremetal", "network-external"),
			},
			nil,
			nil,
		).Once()
}

func expectBaremetalRebuild(mc *cloudmock.MockedCloud, imageID string) {
	mc.Instances.On(
		"BareMetalRebuildInstance",
		mock.Anything,
		testBaremetalID,
		mock.MatchedBy(func(req *edgecloud.BareMetalRebuildRequest) bool {
			return req.ImageID == imageID
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-baremetal-rebuild"}}, nil, nil).Once()

	mc.Tasks.On("Get", mock.Anything, "task-baremetal-rebuild").
		Return(&edgecloud.Task{State: edgecloud.TaskStateFinished}, nil, nil).Once()
}

func baremetalSourceChangeCase(
	name string,
	current map[string]interface{},
	config map[string]interface{},
	installedImageID string,
	rebuildImageID string,
	wantWarning string,
	want map[string]string,
) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)
	if rebuildImageID != "" {
		expectBaremetalRebuild(mc, rebuildImageID)
	}
	gets := 1
	if image, ok := config["image_id"].(string); ok && image != "" && image != current["image_id"] {
		gets = 2
	}
	expectBaremetalRead(mc, baremetalInstance(installedImageID, ""), gets)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:         name,
		Op:           support.OpApply,
		Prepare:      func() *cloudmock.MockedCloud { return mc },
		CurrentID:    testBaremetalID,
		CurrentState: current,
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			if wantWarning == "" {
				support.RequireNoDiags(t, diags)
			} else {
				require.Len(t, diags, 1)
				require.Equal(t, diag.Warning, diags[0].Severity)
				require.Equal(t, "Baremetal instance was not rebuilt", diags[0].Summary)
				require.Contains(t, diags[0].Detail, wantWarning)
			}
			support.RequireStateID(t, state, testBaremetalID)
			support.RequireStateAttrs(t, state, want)
		},
	}
}

func baremetalImageLookupFailureCase(
	name string,
	instance *edgecloud.Instance,
	getErr error,
	wantErr string,
) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)
	mc.Instances.On("Get", mock.Anything, testBaremetalID).Return(instance, nil, getErr).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:         name,
		Op:           support.OpApply,
		Prepare:      func() *cloudmock.MockedCloud { return mc },
		CurrentID:    testBaremetalID,
		CurrentState: baremetalConfig(testBaremetalOldImageID),
		NewConfig:    cloud.Merge(baremetalConfig(testBaremetalNewImageID), cloud.WithName("test-baremetal-renamed")),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			require.True(t, diags.HasError())
			require.Contains(t, diags[0].Summary, wantErr)
			support.RequireStateID(t, state, testBaremetalID)
			support.RequireStateAttrs(t, state, map[string]string{
				"image_id": testBaremetalOldImageID,
				"name":     "test-baremetal",
			})
		},
	}
}

func baremetalReadCase(
	name string,
	current map[string]interface{},
	instance *edgecloud.Instance,
	want map[string]string,
) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)
	expectBaremetalRead(mc, instance, 1)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:         name,
		Op:           support.OpRead,
		Prepare:      func() *cloudmock.MockedCloud { return mc },
		CurrentID:    testBaremetalID,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testBaremetalID)
			support.RequireStateAttrs(t, state, want)
		},
	}
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
		).Times(2)

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

func TestIntegrationBaremetalImageSourceChange(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_baremetal"]
	noSource := map[string]interface{}{}
	appTemplate := map[string]interface{}{"apptemplate_id": testBaremetalAppTemplateID}

	support.RunResourceCases(
		t,
		resource,
		[]support.ResourceCase[*cloudmock.MockedCloud]{
			baremetalSourceChangeCase(
				"unknown installed image does not rebuild",
				baremetalSourceConfig(noSource),
				baremetalConfig(testBaremetalNewImageID),
				"",
				"",
				"The installed image is unknown",
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalSourceChangeCase(
				"image already installed is not rebuilt",
				baremetalSourceConfig(noSource),
				baremetalConfig(testBaremetalNewImageID),
				testBaremetalNewImageID,
				"",
				"",
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalSourceChangeCase(
				"another installed image is rebuilt to the configured one",
				baremetalSourceConfig(noSource),
				baremetalConfig(testBaremetalNewImageID),
				testBaremetalOldImageID,
				testBaremetalNewImageID,
				"",
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalSourceChangeCase(
				"switch to an app template does not rebuild",
				baremetalConfig(testBaremetalOldImageID),
				baremetalSourceConfig(appTemplate),
				"",
				"",
				"The rebuild API accepts only image_id",
				map[string]string{"image_id": "", "apptemplate_id": testBaremetalAppTemplateID},
			),
			baremetalSourceChangeCase(
				"switch from an app template to an image rebuilds",
				baremetalSourceConfig(appTemplate),
				baremetalConfig(testBaremetalNewImageID),
				"",
				testBaremetalNewImageID,
				"",
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalSourceChangeCase(
				"switch from an app template to the installed image does not rebuild",
				baremetalSourceConfig(appTemplate),
				baremetalConfig(testBaremetalNewImageID),
				testBaremetalNewImageID,
				"",
				"The instance already runs this image",
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalImageLookupFailureCase(
				"image lookup failure changes nothing",
				nil,
				errors.New("lookup failed"),
				"lookup failed",
			),
			baremetalImageLookupFailureCase(
				"running task changes nothing",
				baremetalInstance(testBaremetalOldImageID, "task-running"),
				nil,
				"while task task-running is running",
			),
		},
		support.DispatchCase[*cloudmock.MockedCloud],
	)
}

func TestIntegrationBaremetalReadFillsImage(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_baremetal"]

	support.RunResourceCases(
		t,
		resource,
		[]support.ResourceCase[*cloudmock.MockedCloud]{
			baremetalReadCase(
				"imported state takes the image from instance metadata",
				baremetalSourceConfig(map[string]interface{}{}),
				baremetalInstance(testBaremetalOldImageID, ""),
				map[string]string{"image_id": testBaremetalOldImageID},
			),
			baremetalReadCase(
				"state with an image keeps it",
				baremetalConfig(testBaremetalNewImageID),
				baremetalInstance(testBaremetalOldImageID, ""),
				map[string]string{"image_id": testBaremetalNewImageID},
			),
			baremetalReadCase(
				"state with an app template gets no image",
				baremetalSourceConfig(map[string]interface{}{"apptemplate_id": testBaremetalAppTemplateID}),
				baremetalInstance(testBaremetalOldImageID, ""),
				map[string]string{"image_id": "", "apptemplate_id": testBaremetalAppTemplateID},
			),
			baremetalReadCase(
				"state updated by the provider gets no image",
				baremetalSourceConfig(map[string]interface{}{"last_updated": "Tuesday, 27-Oct-26 22:42:49 MSK"}),
				baremetalInstance(testBaremetalOldImageID, ""),
				map[string]string{"image_id": ""},
			),
			baremetalReadCase(
				"instance with a running task gets no image",
				baremetalSourceConfig(map[string]interface{}{}),
				baremetalInstance(testBaremetalOldImageID, "task-running"),
				map[string]string{"image_id": ""},
			),
			baremetalReadCase(
				"metadata without an image leaves the state empty",
				baremetalSourceConfig(map[string]interface{}{}),
				baremetalInstance("", ""),
				map[string]string{"image_id": ""},
			),
		},
		support.DispatchCase[*cloudmock.MockedCloud],
	)
}
