package edgecenter

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceResellerImages() *schema.Resource {
	return &schema.Resource{
		ReadContext:        dataSourceResellerImagesRead,
		DeprecationMessage: "!> **WARNING:** This data source is deprecated and will be removed in the next major version. Use `edgecenter_reseller_imagesV2` data source instead",
		Description: `
**WARNING:** Data source "edgecenter_reseller_images" is deprecated.

Use "edgecenter_reseller_imagesV2" data source instead.`,

		Schema: map[string]*schema.Schema{
			ResellerIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the reseller.",
			},
			ResellerImagesOptionsField: {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "This set defines image IDs that can be attached to the instances of the reseller.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						RegionIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						ImageIDsField: {
							Type:        schema.TypeSet,
							Computed:    true,
							Description: "A list of image IDs available for clients of the reseller.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						CreatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Date when list images was created.",
						},
						UpdatedAtField: {
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
