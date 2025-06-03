package edgecenter

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var errResourceResellerImages = errors.New("resource \"edgecenter_reseller_images\" is deprecated and unavailable")

var ResellerImage = map[string]*schema.Schema{
	RegionIDField: {
		Type:        schema.TypeInt,
		Required:    true,
		Description: "The ID of the region.",
	},
	ImageIDsField: {
		Type:        schema.TypeSet,
		Optional:    true,
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
}

func resourceResellerImages() *schema.Resource {
	return &schema.Resource{
		CreateContext:      resourceResellerImagesCreate,
		ReadContext:        resourceResellerImagesRead,
		UpdateContext:      resourceResellerImagesUpdate,
		DeleteContext:      resourceResellerImagesDelete,
		DeprecationMessage: "!> **WARNING:** This resource is deprecated and will be removed in the next major version. Use `edgecenter_reseller_imagesV2` resource instead",
		Description: `
						**WARNING:** resource "edgecenter_reseller_images" is deprecated.
						Use "edgecenter_reseller_imagesV2" resource instead.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ResellerIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the reseller.",
			},
			ResellerImagesOptionsField: {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "This set defines image IDs that can be attached to the instances of the reseller.",
				Elem: &schema.Resource{
					Schema: ResellerImage,
				},
			},
		},
	}
}

func resourceResellerImagesCreate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(errResourceResellerImages)
}

func resourceResellerImagesRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(errResourceResellerImages)
}

func resourceResellerImagesUpdate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(errResourceResellerImages)
}

func resourceResellerImagesDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(errResourceResellerImages)
}
