package edgecenter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceResellerImagesV2() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceResellerImagesV2Read,
		Description: `
**This resource has been created for resellers and only works with the reseller API key.**

Returns the list of public images currently available for the given project, client, or all clients of a reseller. 
If image_ids = None, all public images are available. If image_ids = [], no public images are available`,

		Schema: map[string]*schema.Schema{
			EntityIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the entity.",
			},
			EntityTypeField: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType}, false),
				Description:  fmt.Sprintf("The entity type. Available values are '%s', '%s', '%s'.", edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType),
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

func dataSourceResellerImagesV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity image reading")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	entityID := d.Get(EntityIDField).(int)
	sntityType := d.Get(EntityTypeField).(string)

	riList, resp, err := clientV2.ResellerImageV2.List(ctx, sntityType, entityID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(entityID))

	riOptions := prepareResellerImagesV2Options(d, riList.Results)

	err = d.Set(ResellerImagesOptionsField, riOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish entity images reading")

	return nil
}
