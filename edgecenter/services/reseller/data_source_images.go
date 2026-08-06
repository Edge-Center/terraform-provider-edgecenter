package reseller

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func dataSourceResellerImages() *schema.Resource {
	return &schema.Resource{
		ReadContext:        dataSourceResellerImagesRead,
		DeprecationMessage: "!> **WARNING:** This data source is deprecated and will be removed in the next major version. Use `edgecenter_reseller_imagesV2` data source instead",
		Description: `
**WARNING:** Data source "edgecenter_reseller_images" is deprecated.

Use "edgecenter_reseller_imagesV2" data source instead.`,

		Schema: map[string]*schema.Schema{
			edgecenter.ResellerIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the reseller.",
			},
			edgecenter.ResellerImagesOptionsField: {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "This set defines image IDs that can be attached to the instances of the reseller.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.RegionIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						edgecenter.ImageIDsField: {
							Type:        schema.TypeSet,
							Computed:    true,
							Description: "A list of image IDs available for clients of the reseller.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						edgecenter.CreatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Date when list images was created.",
						},
						edgecenter.UpdatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Date when list images was last updated.",
						},
					},
				},
			},
		},
	}
}

func dataSourceResellerImagesRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(errResourceResellerImages)
}
