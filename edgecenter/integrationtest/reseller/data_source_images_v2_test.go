//go:build integration

package reseller_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	resellermock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/reseller/mock"
)

const (
	imagesV2DataSourceName = "edgecenter_reseller_imagesV2"

	unsetDataSourceID = "-"

	testOtherRegionID    = 9
	testOtherRegionIDStr = "9"

	firstImageID  = "8e58c8e9-d3d1-4b06-9a3b-b0b0e2a0f0aa"
	secondImageID = "1b1b5c30-8f0e-4a71-9f1a-2b2b2b2b2b2b"

	imagesV2CreatedAt = "2024-01-01T00:00:00Z"
	imagesV2UpdatedAt = "2024-02-02T00:00:00Z"
)

func imagesV2DataSourceState() map[string]interface{} {
	return map[string]interface{}{
		"entity_id":   testEntityID,
		"entity_type": testEntityType,
	}
}

func dataSourceImagesV2List(items ...edgecloud.ResellerImageV2) *edgecloud.ResellerImageV2List {
	return &edgecloud.ResellerImageV2List{Count: len(items), Results: items}
}

func imagesV2WithIDs(regionID int, imageIDs ...string) edgecloud.ResellerImageV2 {
	ids := make(edgecloud.ImageIDs, 0, len(imageIDs))
	ids = append(ids, imageIDs...)

	return edgecloud.ResellerImageV2{
		ImageIDs:   &ids,
		RegionID:   regionID,
		EntityID:   testEntityID,
		EntityType: testEntityType,
		CreatedAt:  imagesV2CreatedAt,
		UpdatedAt:  imagesV2UpdatedAt,
	}
}

func imagesV2WithoutIDs(regionID int) edgecloud.ResellerImageV2 {
	return edgecloud.ResellerImageV2{
		RegionID:   regionID,
		EntityID:   testEntityID,
		EntityType: testEntityType,
		CreatedAt:  imagesV2CreatedAt,
		UpdatedAt:  imagesV2UpdatedAt,
	}
}

func optionPrefix(t *testing.T, state *terraform.InstanceState, regionID string) string {
	t.Helper()

	require.NotNil(t, state, "expected non-nil state")

	for key, value := range state.Attributes {
		if strings.HasPrefix(key, "options.") && strings.HasSuffix(key, ".region_id") && value == regionID {
			return strings.TrimSuffix(key, "region_id")
		}
	}

	t.Fatalf("no options block for region %s, attributes: %v", regionID, state.Attributes)

	return ""
}

func dataSourceOptionImageIDs(state *terraform.InstanceState, prefix string) []string {
	values := make([]string, 0)

	for key, value := range state.Attributes {
		if strings.HasPrefix(key, prefix+"image_ids.") && !strings.HasSuffix(key, ".#") {
			values = append(values, value)
		}
	}

	return values
}

func dataSourceImagesV2ReadCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(dataSourceImagesV2List(
			imagesV2WithIDs(testRegionID, firstImageID, secondImageID),
			imagesV2WithoutIDs(testOtherRegionID),
		), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read stores one options block per region and takes the entity id as the data source id",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				"entity_id":   testEntityIDStr,
				"entity_type": testEntityType,
				"options.#":   "2",
			})

			first := optionPrefix(t, state, testRegionIDStr)
			second := optionPrefix(t, state, testOtherRegionIDStr)
			require.NotEqual(t, first, second)

			support.RequireStateAttrs(t, state, map[string]string{
				first + "created_at":  imagesV2CreatedAt,
				first + "updated_at":  imagesV2UpdatedAt,
				second + "created_at": imagesV2CreatedAt,
				second + "updated_at": imagesV2UpdatedAt,
			})
		},
	}
}

func dataSourceImagesV2ReadImageIDsCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(dataSourceImagesV2List(imagesV2WithIDs(testRegionID, firstImageID, secondImageID)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read keeps all public images unavailable and stores the image ids the api sends",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			prefix := optionPrefix(t, state, testRegionIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				prefix + "all_public_images_are_available": "false",
				prefix + "image_ids.#":                     "2",
			})
			require.ElementsMatch(t,
				[]string{firstImageID, secondImageID},
				dataSourceOptionImageIDs(state, prefix),
			)
		},
	}
}

func dataSourceImagesV2ReadNullImageIDsCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(dataSourceImagesV2List(imagesV2WithoutIDs(testRegionID)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read makes all public images available when the api sends no image id list",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			prefix := optionPrefix(t, state, testRegionIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				prefix + "all_public_images_are_available": "true",
				prefix + "image_ids.#":                     "0",
			})
			require.Empty(t, dataSourceOptionImageIDs(state, prefix))
		},
	}
}

func dataSourceImagesV2ReadEmptyImageIDsCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(dataSourceImagesV2List(imagesV2WithIDs(testRegionID)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read keeps all public images unavailable when the api sends an empty image id list",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			prefix := optionPrefix(t, state, testRegionIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				prefix + "all_public_images_are_available": "false",
				prefix + "image_ids.#":                     "0",
			})
		},
	}
}

func dataSourceImagesV2ReadNotFoundCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil,
			&edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
			fmt.Errorf("entity not found"),
		)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read reports success and sets neither the id nor the options when the api answers not found",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, unsetDataSourceID)
			support.RequireStateAttrs(t, state, map[string]string{
				"options.#": "0",
			})
		},
	}
}

func dataSourceImagesV2ReadAPIFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil,
			&edgecloud.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}},
			fmt.Errorf("api error: server unavailable"),
		)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read surfaces every api error other than not found",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			support.RequireStateID(t, state, unsetDataSourceID)
		},
	}
}

func dataSourceImagesV2ReadWithoutResponseCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := resellermock.NewMockedReseller()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil, nil, fmt.Errorf("api error: connection refused"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         "read surfaces an api error that carries no response",
		Op:           support.OpRead,
		Prepare:      func() *resellermock.MockedReseller { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: imagesV2DataSourceState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "connection refused")
			support.RequireStateID(t, state, unsetDataSourceID)
		},
	}
}

func TestIntegrationResellerImagesV2DataSource_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := resellerDataSource(t, imagesV2DataSourceName)

	cases := []support.ResourceCase[*resellermock.MockedReseller]{
		dataSourceImagesV2ReadCase(),
		dataSourceImagesV2ReadImageIDsCase(),
		dataSourceImagesV2ReadNullImageIDsCase(),
		dataSourceImagesV2ReadEmptyImageIDsCase(),
		dataSourceImagesV2ReadNotFoundCase(),
		dataSourceImagesV2ReadAPIFailureCase(),
		dataSourceImagesV2ReadWithoutResponseCase(),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*resellermock.MockedReseller])
}
