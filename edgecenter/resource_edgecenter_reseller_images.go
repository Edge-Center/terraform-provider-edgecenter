package edgecenter

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

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
		CreateContext: resourceResellerImagesCreate,
		ReadContext:   resourceResellerImagesRead,
		UpdateContext: resourceResellerImagesUpdate,
		DeleteContext: resourceResellerImagesDelete,
		Description: `
					!!! This resource has been created for resellers and only works with the reseller API key. !!!

	Reseller and cloud admin can change the set of images, available to reseller clients.

	Firstly, they may limit the number of public images available.
	Secondly, they can share the image of the reseller client to all clients of the reseller.

	If the reseller has image_ids = [] or hasn't image_ids field in config, 
	all public images are unavailable to the client.`,
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

func resourceResellerImagesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reseller images creating")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	riOptions := d.Get(ResellerImagesOptionsField).(*schema.Set).List()

	sort.Slice(riOptions, func(i, j int) bool {
		iOpt := riOptions[i].(map[string]interface{})
		jOpt := riOptions[j].(map[string]interface{})

		return iOpt[RegionIDField].(int) < jOpt[RegionIDField].(int)
	})

	for _, optRaw := range riOptions {
		imageIDs := edgecloudV2.ImageIDs{}

		opt := optRaw.(map[string]interface{})

		if v, ok := opt[ImageIDsField]; ok {
			imageIDsList := v.(*schema.Set).List()

			imageIDs = make(edgecloudV2.ImageIDs, 0, len(imageIDsList))

			for _, imageID := range imageIDsList {
				imageIDs = append(imageIDs, imageID.(string))
			}
		}

		opts := &edgecloudV2.ResellerImageUpdateRequest{
			ImageIDs:   &imageIDs,
			RegionID:   opt[RegionIDField].(int),
			ResellerID: d.Get(ResellerIDField).(int),
		}

		_, _, err = clientV2.ResellerImage.Update(ctx, opts)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(strconv.Itoa(d.Get(ResellerIDField).(int)))

	resourceResellerImagesRead(ctx, d, m)

	tflog.Debug(ctx, "Finished reseller images creating")

	return nil
}

func resourceResellerImagesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reseller image reading")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	var resellerID int

	if value, ok := d.GetOk(ResellerIDField); ok {
		resellerID = value.(int)
	}

	if value, err := strconv.Atoi(d.Id()); err == nil {
		resellerID = value
	}

	if resellerID == 0 {
		return diag.Errorf("reseller id is empty")
	}

	riList, resp, err := clientV2.ResellerImage.List(ctx, resellerID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return diag.FromErr(err)
	}

	if riList.Count == 0 {
		return nil
	}

	sort.Slice(riList.Results, func(i, j int) bool {
		return riList.Results[i].RegionID < riList.Results[j].RegionID
	})

	err = d.Set(ResellerIDField, resellerID)
	if err != nil {
		return diag.FromErr(err)
	}

	riOptions := prepareResellerImagesOptions(d, riList.Results)

	err = d.Set(ResellerImagesOptionsField, riOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reseller images reading")

	return nil
}

func resourceResellerImagesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "start reseller image updating")

	err := resourceResellerImagesDelete(ctx, d, m)
	if err != nil {
		rollbackResellerImagesData(ctx, d)
		return resourceResellerImagesCreate(ctx, d, m)
	}

	err = resourceResellerImagesCreate(ctx, d, m)
	if err != nil {
		rollbackResellerImagesData(ctx, d)
		return resourceResellerImagesCreate(ctx, d, m)
	}

	d.SetId(strconv.Itoa(d.Get(ResellerIDField).(int)))

	tflog.Debug(ctx, "finish reseller images updating")

	return nil
}

func resourceResellerImagesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "start reseller images deleting")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	resellerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = clientV2.ResellerImage.Delete(ctx, resellerID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Debug(ctx, "finish reseller images deleting")

	return nil
}
