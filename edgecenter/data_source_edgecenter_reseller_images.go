package edgecenter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceResellerImages() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceResellerImagesRead,
		Description: `
					!!! This resource has been created for resellers and only works with the reseller API key. !!!

		Reseller and cloud admin can change the set of images, available to reseller clients.

		Firstly, they may limit the number of public images available.
		Secondly, they can share the image of the reseller client to all clients of the reseller.

		If the reseller has image_ids = [] or hasn't image_ids field in config, 
		all public images are unavailable to the client.`,

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

func dataSourceResellerImagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reseller image reading")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	resellerID := d.Get(ResellerIDField).(int)

	riList, resp, err := clientV2.ResellerImage.List(ctx, resellerID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(resellerID))

	riOptions := prepareResellerImagesOptions(d, riList.Results)

	err = d.Set(ResellerImagesOptionsField, riOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reseller images reading")

	return nil
}
