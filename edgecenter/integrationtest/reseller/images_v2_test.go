//go:build integration

package reseller_test

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	resellermock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/reseller/mock"
	resellersvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/reseller"
)

const (
	testImageIDA = "11111111-1111-1111-1111-111111111111"
	testImageIDB = "22222222-2222-2222-2222-222222222222"

	testLowerRegionID    = 3
	testLowerRegionIDStr = "3"

	testCreatedAt = "2024-01-01T00:00:00Z"
	testUpdatedAt = "2024-02-02T00:00:00Z"
)

func imagesV2Mock() *resellermock.MockedReseller {
	return resellermock.NewMockedReseller()
}

func imagesV2OptionsBlock(regionID int, imageIDs []interface{}, allPublic bool) map[string]interface{} {
	return map[string]interface{}{
		edgecenter.RegionIDField:                    regionID,
		edgecenter.ImageIDsField:                    imageIDs,
		edgecenter.AllPublicImagesAreAvailableField: allPublic,
	}
}

func imagesV2Config(blocks ...interface{}) map[string]interface{} {
	return map[string]interface{}{
		edgecenter.EntityIDField:              testEntityID,
		edgecenter.EntityTypeField:            testEntityType,
		edgecenter.ResellerImagesOptionsField: blocks,
	}
}

func imagesV2Remote(regionID int, imageIDs *edgecloud.ImageIDs, createdAt, updatedAt string) edgecloud.ResellerImageV2 {
	return edgecloud.ResellerImageV2{
		ImageIDs:   imageIDs,
		RegionID:   regionID,
		EntityID:   testEntityID,
		EntityType: testEntityType,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

func imagesV2List(items ...edgecloud.ResellerImageV2) *edgecloud.ResellerImageV2List {
	return &edgecloud.ResellerImageV2List{Count: len(items), Results: items}
}

func captureUpdateRequests(mc *resellermock.MockedReseller, sent *[]*edgecloud.ResellerImageV2UpdateRequest) {
	mc.Images.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			*sent = append(*sent, args.Get(1).(*edgecloud.ResellerImageV2UpdateRequest))
		}).
		Return(&edgecloud.ResellerImageV2{}, nil, nil)
}

func imagesV2RawConfig(config map[string]interface{}) cty.Value {
	blocks, _ := config[edgecenter.ResellerImagesOptionsField].([]interface{})

	values := make([]cty.Value, 0, len(blocks))
	for _, raw := range blocks {
		block := raw.(map[string]interface{})

		imageIDs := cty.NullVal(cty.Set(cty.String))
		switch ids, ok := block[edgecenter.ImageIDsField].([]interface{}); {
		case !ok || ids == nil:
		case len(ids) == 0:
			imageIDs = cty.SetValEmpty(cty.String)
		default:
			items := make([]cty.Value, 0, len(ids))
			for _, id := range ids {
				items = append(items, cty.StringVal(id.(string)))
			}
			imageIDs = cty.SetVal(items)
		}

		values = append(values, cty.ObjectVal(map[string]cty.Value{
			edgecenter.ImageIDsField:                    imageIDs,
			edgecenter.AllPublicImagesAreAvailableField: cty.BoolVal(block[edgecenter.AllPublicImagesAreAvailableField].(bool)),
		}))
	}

	return cty.ObjectVal(map[string]cty.Value{
		edgecenter.ResellerImagesOptionsField: cty.TupleVal(values),
	})
}

func imagesV2Runner(
	t *testing.T,
	resource *schema.Resource,
	tc support.ResourceCase[*resellermock.MockedReseller],
	fake *resellermock.MockedReseller,
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	if tc.Op != support.OpApply {
		return support.DispatchCase(t, resource, tc, fake)
	}

	ctx := context.Background()
	meta := fake.TestMeta()

	current := &terraform.InstanceState{Attributes: map[string]string{}}
	if tc.CurrentState != nil || tc.CurrentID != "" {
		current = support.NewState(t, resource, tc.CurrentState, tc.CurrentID)
	}
	current.RawConfig = imagesV2RawConfig(tc.NewConfig)

	diff, err := resource.Diff(ctx, current, terraform.NewResourceConfigRaw(tc.NewConfig), meta)
	require.NoError(t, err)

	return resource.Apply(ctx, current, diff, meta)
}

func optionValues(state *terraform.InstanceState, field string) []string {
	prefix := edgecenter.ResellerImagesOptionsField + "."
	suffix := "." + field

	values := make([]string, 0)
	for key, value := range state.Attributes {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			values = append(values, value)
		}
	}
	sort.Strings(values)

	return values
}

func optionImageIDs(state *terraform.InstanceState) []string {
	prefix := edgecenter.ResellerImagesOptionsField + "."
	marker := "." + edgecenter.ImageIDsField + "."

	imageIDs := make([]string, 0)
	for key, value := range state.Attributes {
		if !strings.HasPrefix(key, prefix) || !strings.Contains(key, marker) || strings.HasSuffix(key, ".#") {
			continue
		}
		imageIDs = append(imageIDs, value)
	}
	sort.Strings(imageIDs)

	return imageIDs
}

func imagesV2CreateCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	captureUpdateRequests(mc, &sent)

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDA, testImageIDB}), "", "")), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create sends one update request per options block and stores the entity id as the resource id",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA, testImageIDB}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			require.Len(t, sent, 1)
			require.Equal(t, testEntityID, sent[0].EntityID)
			require.Equal(t, testEntityType, sent[0].EntityType)
			require.Equal(t, testRegionID, sent[0].RegionID)
			require.NotNil(t, sent[0].ImageIDs)
			require.ElementsMatch(t, []string{testImageIDA, testImageIDB}, []string(*sent[0].ImageIDs))

			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.EntityIDField:                     testEntityIDStr,
				edgecenter.EntityTypeField:                   testEntityType,
				edgecenter.ResellerImagesOptionsField + ".#": "1",
			})
			require.Equal(t, []string{testRegionIDStr}, optionValues(state, edgecenter.RegionIDField))
			require.Equal(t, []string{testImageIDA, testImageIDB}, optionImageIDs(state))
		},
	}
}

func imagesV2CreateSeveralOptionsBlocksCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	captureUpdateRequests(mc, &sent)

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(
			imagesV2Remote(testLowerRegionID, ptr(edgecloud.ImageIDs{testImageIDB}), "", ""),
			imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDA}), "", ""),
		), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create walks the options blocks in ascending region id order",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
			imagesV2OptionsBlock(testLowerRegionID, []interface{}{testImageIDB}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			require.Len(t, sent, 2)
			require.Equal(t, testLowerRegionID, sent[0].RegionID)
			require.Equal(t, testRegionID, sent[1].RegionID)
			require.ElementsMatch(t, []string{testImageIDB}, []string(*sent[0].ImageIDs))
			require.ElementsMatch(t, []string{testImageIDA}, []string(*sent[1].ImageIDs))

			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.ResellerImagesOptionsField + ".#": "2",
			})
			require.Equal(t, []string{testLowerRegionIDStr, testRegionIDStr}, optionValues(state, edgecenter.RegionIDField))
		},
	}
}

func imagesV2CreateAllPublicImagesCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	captureUpdateRequests(mc, &sent)

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, nil, "", "")), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create sends a nil image id list when all public images are available",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, nil, true),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			require.Len(t, sent, 1)
			require.Nil(t, sent[0].ImageIDs)
			require.Equal(t, testRegionID, sent[0].RegionID)

			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.ResellerImagesOptionsField + ".#": "1",
			})
			require.Equal(t, []string{"true"}, optionValues(state, edgecenter.AllPublicImagesAreAvailableField))
			require.Empty(t, optionImageIDs(state))
		},
	}
}

func imagesV2CreateAPIFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("Update", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: image is not public"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create surfaces the api error",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "image is not public")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Images.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func imagesV2CreateSwallowsReadFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("Update", mock.Anything, mock.Anything).
		Return(&edgecloud.ResellerImageV2{}, nil, nil)
	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create reports success when the trailing read fails because the read diagnostics are dropped",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
		},
	}
}

func imagesV2CreateKeepsConfiguredBlockCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("Update", mock.Anything, mock.Anything).
		Return(&edgecloud.ResellerImageV2{}, nil, nil)
	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDB}), testCreatedAt, testUpdatedAt)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:    "create keeps the configured block next to the one the api reports because the read adds to the options set instead of replacing it",
		Op:      support.OpApply,
		Prepare: func() *resellermock.MockedReseller { return mc },
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.ResellerImagesOptionsField + ".#": "2",
			})
			require.Equal(t, []string{testRegionIDStr, testRegionIDStr}, optionValues(state, edgecenter.RegionIDField))
			require.Equal(t, []string{"", testCreatedAt}, optionValues(state, edgecenter.CreatedAtField))
			require.Equal(t, []string{"", testUpdatedAt}, optionValues(state, edgecenter.UpdatedAtField))
			require.Equal(t, []string{testImageIDA, testImageIDB}, optionImageIDs(state))
		},
	}
}

func imagesV2ReadCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDA, testImageIDB}), testCreatedAt, testUpdatedAt)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read stores the entity id, the entity type and the options the api reports",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: map[string]interface{}{
			edgecenter.EntityTypeField:            testEntityType,
			edgecenter.ResellerImagesOptionsField: []interface{}{},
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.EntityIDField:                     testEntityIDStr,
				edgecenter.EntityTypeField:                   testEntityType,
				edgecenter.ResellerImagesOptionsField + ".#": "1",
			})
			require.Equal(t, []string{testRegionIDStr}, optionValues(state, edgecenter.RegionIDField))
			require.Equal(t, []string{testCreatedAt}, optionValues(state, edgecenter.CreatedAtField))
			require.Equal(t, []string{testUpdatedAt}, optionValues(state, edgecenter.UpdatedAtField))
			require.Equal(t, []string{testImageIDA, testImageIDB}, optionImageIDs(state))
		},
	}
}

func imagesV2ReadDropsRefreshedTimestampsCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDA}), testCreatedAt, testUpdatedAt)), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read drops the timestamps the api reports because the block already in state wins the set insert",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.ResellerImagesOptionsField + ".#": "1",
			})
			require.Equal(t, []string{""}, optionValues(state, edgecenter.CreatedAtField))
			require.Equal(t, []string{""}, optionValues(state, edgecenter.UpdatedAtField))
		},
	}
}

func imagesV2ReadNotFoundCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read reports no error when the api answers 404",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
		},
	}
}

func imagesV2ReadEmptyListCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read leaves the options untouched when the api reports no entries",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateID(t, state, testEntityIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				edgecenter.ResellerImagesOptionsField + ".#": "1",
			})
			require.Equal(t, []string{testImageIDA}, optionImageIDs(state))
		},
	}
}

func imagesV2ReadEmptyEntityTypeCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read rejects an empty entity type",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: map[string]interface{}{
			edgecenter.EntityIDField: testEntityID,
			edgecenter.ResellerImagesOptionsField: []interface{}{
				imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
			},
		},
		Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, fake *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entity type is empty")
			fake.Images.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func imagesV2ReadAPIFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, fmt.Errorf("api error: backend is down"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "read surfaces the api error and keeps the id in state",
		Op:        support.OpRead,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "backend is down")
			support.RequireStateID(t, state, testEntityIDStr)
		},
	}
}

func imagesV2UpdateCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	var calls []string

	mc.Images.On("Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil)).
		Run(func(_ mock.Arguments) { calls = append(calls, "delete") }).
		Return(nil, nil)
	mc.Images.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			calls = append(calls, "update")
			sent = append(sent, args.Get(1).(*edgecloud.ResellerImageV2UpdateRequest))
		}).
		Return(&edgecloud.ResellerImageV2{}, nil, nil)
	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Run(func(_ mock.Arguments) { calls = append(calls, "list") }).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDB}), "", "")), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "update deletes the entity and recreates it from the new options",
		Op:        support.OpApply,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDB}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)

			fake.Images.AssertCalled(t, "Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil))
			require.Equal(t, []string{"delete", "update", "list"}, calls)
			require.Len(t, sent, 1)
			require.ElementsMatch(t, []string{testImageIDB}, []string(*sent[0].ImageIDs))

			support.RequireStateID(t, state, testEntityIDStr)
			require.Equal(t, []string{testImageIDB}, optionImageIDs(state))
		},
	}
}

func imagesV2UpdateDeleteFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	mc.Images.On("Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil)).
		Return(nil, fmt.Errorf("api error: entity is locked"))
	captureUpdateRequests(mc, &sent)
	mc.Images.On("List", mock.Anything, testEntityType, testEntityID).
		Return(imagesV2List(imagesV2Remote(testRegionID, ptr(edgecloud.ImageIDs{testImageIDA}), "", "")), nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "update pushes the previous options back to the api after a failed delete and still reports an error",
		Op:        support.OpApply,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDB}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "deleting error while reseller images update")

			require.Len(t, sent, 1)
			require.ElementsMatch(t, []string{testImageIDA}, []string(*sent[0].ImageIDs))

			support.RequireStateID(t, state, testEntityIDStr)
		},
	}
}

func imagesV2UpdateCreateFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	var sent []*edgecloud.ResellerImageV2UpdateRequest

	mc.Images.On("Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil)).
		Return(nil, nil)
	mc.Images.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			sent = append(sent, args.Get(1).(*edgecloud.ResellerImageV2UpdateRequest))
		}).
		Return(nil, nil, fmt.Errorf("api error: image is not public"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "update replays the failing create with the previous options and leaves the entity type in the resource id",
		Op:        support.OpApply,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		NewConfig: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDB}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "creating error while reseller images update")

			require.Len(t, sent, 2)
			require.ElementsMatch(t, []string{testImageIDB}, []string(*sent[0].ImageIDs))
			require.ElementsMatch(t, []string{testImageIDA}, []string(*sent[1].ImageIDs))

			support.RequireStateID(t, state, testEntityType)
		},
	}
}

func imagesV2DeleteCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil)).
		Return(nil, nil)

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "delete calls the api with the entity type and id and clears the resource id",
		Op:        support.OpDelete,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *resellermock.MockedReseller) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
			fake.Images.AssertCalled(t, "Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil))
		},
	}
}

func imagesV2DeleteAPIFailureCase() support.ResourceCase[*resellermock.MockedReseller] {
	mc := imagesV2Mock()

	mc.Images.On("Delete", mock.Anything, testEntityType, testEntityID, (*edgecloud.ResellerImageV2DeleteOptions)(nil)).
		Return(nil, fmt.Errorf("api error: entity is locked"))

	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:      "delete surfaces the api error and keeps the id in state",
		Op:        support.OpDelete,
		Prepare:   func() *resellermock.MockedReseller { return mc },
		CurrentID: testEntityIDStr,
		CurrentState: imagesV2Config(
			imagesV2OptionsBlock(testRegionID, []interface{}{testImageIDA}, false),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *resellermock.MockedReseller) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entity is locked")
			support.RequireStateID(t, state, testEntityIDStr)
		},
	}
}

func TestIntegrationResellerImagesV2_TableDriven(t *testing.T) {
	t.Parallel()

	resource := resellerResource(t, resellersvc.ResellerImagesV2Resource)

	cases := []support.ResourceCase[*resellermock.MockedReseller]{
		imagesV2CreateCase(),
		imagesV2CreateSeveralOptionsBlocksCase(),
		imagesV2CreateAllPublicImagesCase(),
		imagesV2CreateAPIFailureCase(),
		imagesV2CreateSwallowsReadFailureCase(),
		imagesV2CreateKeepsConfiguredBlockCase(),
		imagesV2ReadCase(),
		imagesV2ReadDropsRefreshedTimestampsCase(),
		imagesV2ReadNotFoundCase(),
		imagesV2ReadEmptyListCase(),
		imagesV2ReadEmptyEntityTypeCase(),
		imagesV2ReadAPIFailureCase(),
		imagesV2UpdateCase(),
		imagesV2UpdateDeleteFailureCase(),
		imagesV2UpdateCreateFailureCase(),
		imagesV2DeleteCase(),
		imagesV2DeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, imagesV2Runner)
}

func TestIntegrationResellerImagesV2Importer(t *testing.T) {
	t.Parallel()

	resource := resellerResource(t, resellersvc.ResellerImagesV2Resource)
	require.NotNil(t, resource.Importer)
	require.NotNil(t, resource.Importer.StateContext)

	t.Run("import splits the entity type from the entity id and keeps the id numeric", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(&terraform.InstanceState{ID: testEntityType + ":" + testEntityIDStr})

		results, err := resource.Importer.StateContext(context.Background(), data, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, testEntityType, results[0].Get(edgecenter.EntityTypeField))
		require.Equal(t, testEntityID, results[0].Get(edgecenter.EntityIDField))
		require.Equal(t, testEntityIDStr, results[0].Id())
	})

	rejected := []struct {
		name string
		id   string
	}{
		{name: "import rejects an id without a separator", id: testEntityIDStr},
		{name: "import rejects an unknown entity type", id: "partner:" + testEntityIDStr},
		{name: "import rejects a non numeric entity id", id: testEntityType + ":none"},
	}

	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := resource.Data(&terraform.InstanceState{ID: tc.id})

			results, err := resource.Importer.StateContext(context.Background(), data, nil)
			require.Error(t, err)
			require.Nil(t, results)
		})
	}
}
