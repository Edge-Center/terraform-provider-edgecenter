package edgecenter

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

var ResellerImageV2 = map[string]*schema.Schema{
	RegionIDField: {
		Type:        schema.TypeInt,
		Required:    true,
		Description: "The ID of the region.",
	},
	ImageIDsField: {
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "A list of image IDs available for clients of the entity.",
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

func resourceResellerImagesV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceResellerImagesV2Create,
		ReadContext:   resourceResellerImagesV2Read,
		UpdateContext: resourceResellerImagesV2Update,
		DeleteContext: resourceResellerImagesV2Delete,
		Description: `
**This resource has been created for resellers and only works with the reseller API key.**

Resellers and Cloud Admins can change the set of images available to resellers, their customers and their projects.

Firstly, they can limit the number of public images available.
If the reseller, client or project has image_ids = [] or doesn't have an image_ids field in config, all public images will be unavailable to the client.`,
		Schema: map[string]*schema.Schema{
			EntityIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the entity.",
			},
			EntityTypeField: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType}, false),
				Description:  fmt.Sprintf("The entity type. Available values are '%s', '%s', '%s'.", edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType),
			},
			ResellerImagesOptionsField: {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "This set defines image IDs that can be attached to the instances of the entity.",
				Elem: &schema.Resource{
					Schema: ResellerImageV2,
				},
			},
		},
	}
}

func resourceResellerImagesV2Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity images creating")

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

		opts := &edgecloudV2.ResellerImageV2UpdateRequest{
			ImageIDs:   &imageIDs,
			RegionID:   opt[RegionIDField].(int),
			EntityID:   d.Get(EntityIDField).(int),
			EntityType: d.Get(EntityTypeField).(string),
		}

		_, _, err = clientV2.ResellerImageV2.Update(ctx, opts)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(strconv.Itoa(d.Get(EntityIDField).(int)))

	resourceResellerImagesV2Read(ctx, d, m)

	tflog.Debug(ctx, "Finished entity images creating")

	return nil
}

func resourceResellerImagesV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity image reading")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	var entityID int

	if value, ok := d.GetOk(EntityIDField); ok {
		entityID = value.(int)
	}

	if value, err := strconv.Atoi(d.Id()); err == nil {
		entityID = value
	}

	if entityID == 0 {
		return diag.Errorf("entity ID is empty")
	}

	entityType := d.Get(EntityTypeField).(string)

	if entityType == "" {
		return diag.Errorf("entity type is empty")
	}

	riList, resp, err := clientV2.ResellerImageV2.List(ctx, entityType, entityID)
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

	err = d.Set(EntityIDField, entityID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(EntityTypeField, entityType)
	if err != nil {
		return diag.FromErr(err)
	}

	riOptions := prepareResellerImagesV2Options(d, riList.Results)

	err = d.Set(ResellerImagesOptionsField, riOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish entity images reading")

	return nil
}

func resourceResellerImagesV2Update(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity image updating")

	err := resourceResellerImagesV2Delete(ctx, d, m)
	if err != nil {
		rollbackResellerImagesV2Data(ctx, d)
		resourceResellerImagesV2Create(ctx, d, m)
		return diag.Errorf("deleting error while reseller images update: %s", DiagnosticsToString(err))
	}

	err = resourceResellerImagesV2Create(ctx, d, m)
	if err != nil {
		rollbackResellerImagesV2Data(ctx, d)
		resourceResellerImagesV2Create(ctx, d, m)
		return diag.Errorf("creating error while reseller images update: %s", DiagnosticsToString(err))
	}

	d.SetId(strconv.Itoa(d.Get(EntityIDField).(int)))

	tflog.Debug(ctx, "Finish entity images updating")

	return nil
}

func resourceResellerImagesV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity images deleting")

	clientV2, err := InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	entityID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	entityType := d.Get(EntityTypeField).(string)

	if entityType == "" {
		return diag.Errorf("entity type is empty")
	}

	_, err = clientV2.ResellerImageV2.Delete(ctx, entityType, entityID, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Debug(ctx, "Finish entity images deleting")

	return nil
}
