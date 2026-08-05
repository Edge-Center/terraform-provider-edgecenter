package reseller

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	ResellerImagesV2Resource   = "edgecenter_reseller_imagesV2"
	ResellerImagesV2DataSource = "edgecenter_reseller_imagesV2"
)

var ResellerImageV2 = map[string]*schema.Schema{
	edgecenter.RegionIDField: {
		Type:        schema.TypeInt,
		Required:    true,
		Description: "The ID of the region.",
	},
	edgecenter.ImageIDsField: {
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "A list of image IDs available for clients of the entity.",
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	},
	edgecenter.AllPublicImagesAreAvailableField: {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Flag to indicate that all public images are available.",
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
}

func resourceResellerImagesV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceResellerImagesV2Create,
		ReadContext:   resourceResellerImagesV2Read,
		UpdateContext: resourceResellerImagesV2Update,
		DeleteContext: resourceResellerImagesV2Delete,
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), ":")
				if len(parts) != 2 {
					return nil, fmt.Errorf("failed import: id must have format <entity_type>:<entity_id>")
				}
				entityType := parts[0]
				switch entityType {
				case edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType:
				default:
					return nil, fmt.Errorf("failed import: entity_type must be one of '%s', '%s', '%s'",
						edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType)
				}
				entityID, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, fmt.Errorf("failed import: entity_id must be a number: %w", err)
				}
				if err := d.Set(edgecenter.EntityTypeField, entityType); err != nil {
					return nil, fmt.Errorf("failed import: %w", err)
				}
				if err := d.Set(edgecenter.EntityIDField, entityID); err != nil {
					return nil, fmt.Errorf("failed import: %w", err)
				}
				d.SetId(parts[1])

				return []*schema.ResourceData{d}, nil
			},
		},
		Description: `
**This resource has been created for resellers and only works with the reseller API key.**

Resellers and Cloud Admins can change the set of images available to resellers, their customers and their projects.

Firstly, they can limit the number of public images available.
If the reseller, client or project has image_ids = [] or doesn't have an image_ids field in config, all public images will be unavailable to the client.`,
		Schema: map[string]*schema.Schema{
			edgecenter.EntityIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the entity.",
			},
			edgecenter.EntityTypeField: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType}, false),
				Description:  fmt.Sprintf("The entity type. Available values are '%s', '%s', '%s'.", edgecloudV2.ResellerType, edgecloudV2.ClientType, edgecloudV2.ProjectType),
			},
			edgecenter.ResellerImagesOptionsField: {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "This set defines image IDs that can be attached to the instances of the entity.",
				Elem: &schema.Resource{
					Schema: ResellerImageV2,
				},
			},
		},
		CustomizeDiff: customdiff.All(
			validateResellerImagesOptions,
		),
	}
}

func validateResellerImagesOptions(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	rawOptionsConfig := diff.GetRawConfig().GetAttr(edgecenter.ResellerImagesOptionsField)
	rawOptionsList := rawOptionsConfig.AsValueSlice()

	for _, val := range rawOptionsList {
		isImageIDsNull := val.GetAttr(edgecenter.ImageIDsField).IsNull()
		areAllPublicImagesAvailable := val.GetAttr(edgecenter.AllPublicImagesAreAvailableField).True()
		if !isImageIDsNull && areAllPublicImagesAvailable {
			return fmt.Errorf(
				"%s must not be set when %s is true",
				edgecenter.ImageIDsField,
				edgecenter.AllPublicImagesAreAvailableField,
			)
		}
	}

	return nil
}

func resourceResellerImagesV2Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity images creating")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	riOptions := d.Get(edgecenter.ResellerImagesOptionsField).(*schema.Set).List()

	sort.Slice(riOptions, func(i, j int) bool {
		iOpt := riOptions[i].(map[string]interface{})
		jOpt := riOptions[j].(map[string]interface{})

		return iOpt[edgecenter.RegionIDField].(int) < jOpt[edgecenter.RegionIDField].(int)
	})

	for _, optRaw := range riOptions {
		opt := optRaw.(map[string]interface{})

		areAllPublicImagesAvailable := false
		if v, ok := opt[edgecenter.AllPublicImagesAreAvailableField]; ok {
			areAllPublicImagesAvailable = v.(bool)
		}

		var imageIDsPtr *edgecloudV2.ImageIDs = nil

		if !areAllPublicImagesAvailable {
			imageIDs := edgecloudV2.ImageIDs{}
			if v, ok := opt[edgecenter.ImageIDsField]; ok {
				imageIDsList := v.(*schema.Set).List()
				imageIDs = make(edgecloudV2.ImageIDs, 0, len(imageIDsList))
				for _, imageID := range imageIDsList {
					imageIDs = append(imageIDs, imageID.(string))
				}
			}
			imageIDsPtr = &imageIDs
		}

		opts := &edgecloudV2.ResellerImageV2UpdateRequest{
			ImageIDs:   imageIDsPtr,
			RegionID:   opt[edgecenter.RegionIDField].(int),
			EntityID:   d.Get(edgecenter.EntityIDField).(int),
			EntityType: d.Get(edgecenter.EntityTypeField).(string),
		}

		_, _, err = clientV2.ResellerImageV2.Update(ctx, opts)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(strconv.Itoa(d.Get(edgecenter.EntityIDField).(int)))

	resourceResellerImagesV2Read(ctx, d, m)

	tflog.Debug(ctx, "Finished entity images creating")

	return nil
}

func resourceResellerImagesV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity image reading")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	var entityID int

	if value, ok := d.GetOk(edgecenter.EntityIDField); ok {
		entityID = value.(int)
	}

	if value, err := strconv.Atoi(d.Id()); err == nil {
		entityID = value
	}

	if entityID == 0 {
		return diag.Errorf("entity ID is empty")
	}

	entityType := d.Get(edgecenter.EntityTypeField).(string)

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

	err = d.Set(edgecenter.EntityIDField, entityID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(edgecenter.EntityTypeField, entityType)
	if err != nil {
		return diag.FromErr(err)
	}

	riOptions := prepareResellerImagesV2Options(d, riList.Results)

	err = d.Set(edgecenter.ResellerImagesOptionsField, riOptions)
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
		return diag.Errorf("deleting error while reseller images update: %s", edgecenter.DiagnosticsToString(err))
	}

	err = resourceResellerImagesV2Create(ctx, d, m)
	if err != nil {
		rollbackResellerImagesV2Data(ctx, d)
		resourceResellerImagesV2Create(ctx, d, m)
		return diag.Errorf("creating error while reseller images update: %s", edgecenter.DiagnosticsToString(err))
	}

	d.SetId(strconv.Itoa(d.Get(edgecenter.EntityIDField).(int)))

	tflog.Debug(ctx, "Finish entity images updating")

	return nil
}

func resourceResellerImagesV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start entity images deleting")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, resellerImagesCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	entityID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	entityType := d.Get(edgecenter.EntityTypeField).(string)

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

func resellerImagesCloudClientConf() *edgecenter.CloudClientConf {
	return &edgecenter.CloudClientConf{
		DoNotUseProjectID: true,
		DoNotUseRegionID:  true,
	}
}

func rollbackResellerImagesV2Data(ctx context.Context, d *schema.ResourceData) {
	resellerImagesOptions, _ := d.GetChange(edgecenter.ResellerImagesOptionsField)
	err := d.Set(edgecenter.ResellerImagesOptionsField, resellerImagesOptions)
	if err != nil {
		tflog.Error(ctx, "set old \"image_ids\" error: "+err.Error())
	}

	oldEntityID, _ := d.GetChange(edgecenter.EntityIDField)
	d.SetId(strconv.Itoa(oldEntityID.(int)))

	oldEntityType, _ := d.GetChange(edgecenter.EntityTypeField)
	d.SetId(oldEntityType.(string))
}

func prepareResellerImagesV2Options(d *schema.ResourceData, riList []edgecloudV2.ResellerImageV2) *schema.Set {
	riOptions := d.Get(edgecenter.ResellerImagesOptionsField).(*schema.Set)

	for _, ri := range riList {
		riOption := make(map[string]interface{})

		if ri.ImageIDs != nil {
			imageIDs := make([]interface{}, 0, len(*ri.ImageIDs))

			for _, imageID := range *ri.ImageIDs {
				imageIDs = append(imageIDs, imageID)
			}

			riOption[edgecenter.ImageIDsField] = schema.NewSet(schema.HashString, imageIDs)
			riOption[edgecenter.AllPublicImagesAreAvailableField] = false
		} else {
			riOption[edgecenter.AllPublicImagesAreAvailableField] = true
		}

		riOption[edgecenter.RegionIDField] = ri.RegionID
		riOption[edgecenter.CreatedAtField] = ri.CreatedAt
		riOption[edgecenter.UpdatedAtField] = ri.UpdatedAt

		riOptions.Add(riOption)
	}

	return riOptions
}
