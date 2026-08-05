//go:build integration

package reseller_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	resellermock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/reseller/mock"
	resellersvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/reseller"
)

const (
	deprecatedImagesRefusal     = `resource "edgecenter_reseller_images" is deprecated and unavailable`
	deprecatedImagesReplacement = "edgecenter_reseller_imagesV2"
)

func deprecatedImagesConfig(imageID string) map[string]interface{} {
	return map[string]interface{}{
		edgecenter.ResellerIDField: testEntityID,
		edgecenter.ResellerImagesOptionsField: []interface{}{
			map[string]interface{}{
				edgecenter.RegionIDField: testRegionID,
				edgecenter.ImageIDsField: []interface{}{imageID},
			},
		},
	}
}

func deprecatedImagesCase(
	name string,
	op support.Operation,
	currentID string,
	currentState map[string]interface{},
	newConfig map[string]interface{},
) support.ResourceCase[*resellermock.MockedReseller] {
	return support.ResourceCase[*resellermock.MockedReseller]{
		Name:         name,
		Op:           op,
		Prepare:      func() *resellermock.MockedReseller { return resellermock.NewMockedReseller() },
		CurrentID:    currentID,
		CurrentState: currentState,
		NewConfig:    newConfig,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *resellermock.MockedReseller) {
			require.Len(t, diags, 1, "the refusal must be the only diagnostic")
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, deprecatedImagesRefusal)

			if currentID == "" {
				require.Nil(t, state, "a refused create must not produce state")
			} else {
				support.RequireStateID(t, state, currentID)
			}

			require.Empty(t, fake.Images.Calls, "the deprecated pair must not reach the cloud sdk")
			require.Empty(t, fake.Networks.Calls, "the deprecated pair must not reach the cloud sdk")
		},
	}
}

func TestIntegrationResellerImagesDeprecatedResource_TableDriven(t *testing.T) {
	t.Parallel()

	resource := resellerResource(t, resellersvc.ResellerImagesResource)

	cases := []support.ResourceCase[*resellermock.MockedReseller]{
		deprecatedImagesCase(
			"create refuses with the deprecation error and never reaches the cloud api",
			support.OpApply,
			"",
			nil,
			deprecatedImagesConfig("image-a"),
		),
		deprecatedImagesCase(
			"read refuses with the deprecation error and never reaches the cloud api",
			support.OpRead,
			testEntityIDStr,
			deprecatedImagesConfig("image-a"),
			nil,
		),
		deprecatedImagesCase(
			"update refuses with the deprecation error and never reaches the cloud api",
			support.OpApply,
			testEntityIDStr,
			deprecatedImagesConfig("image-a"),
			deprecatedImagesConfig("image-b"),
		),
		deprecatedImagesCase(
			"delete refuses with the deprecation error and never reaches the cloud api",
			support.OpDelete,
			testEntityIDStr,
			deprecatedImagesConfig("image-a"),
			nil,
		),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*resellermock.MockedReseller])
}

func TestIntegrationResellerImagesDeprecatedDataSource_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := resellerDataSource(t, resellersvc.ResellerImagesDataSource)

	cases := []support.ResourceCase[*resellermock.MockedReseller]{
		deprecatedImagesCase(
			"read refuses with the resource-worded deprecation error although this is a data source",
			support.OpRead,
			testEntityIDStr,
			map[string]interface{}{edgecenter.ResellerIDField: testEntityID},
			nil,
		),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*resellermock.MockedReseller])
}

func TestIntegrationResellerImagesDeprecated_PointsAtTheV2Replacement(t *testing.T) {
	t.Parallel()

	targets := map[string]*schema.Resource{
		"resource":    resellerResource(t, resellersvc.ResellerImagesResource),
		"data source": resellerDataSource(t, resellersvc.ResellerImagesDataSource),
	}

	for kind, target := range targets {
		t.Run(kind, func(t *testing.T) {
			require.Contains(t, target.DeprecationMessage, deprecatedImagesReplacement)
		})
	}
}
