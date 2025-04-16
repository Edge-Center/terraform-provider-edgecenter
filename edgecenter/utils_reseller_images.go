package edgecenter

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func resellerImagesCloudClientConf() *CloudClientConf {
	return &CloudClientConf{
		DoNotUseProjectID: true,
		DoNotUseRegionID:  true,
	}
}

func rollbackResellerImagesData(ctx context.Context, d *schema.ResourceData) {
	resellerImagesOptions, _ := d.GetChange(ResellerImagesOptionsField)
	err := d.Set(ResellerImagesOptionsField, resellerImagesOptions)
	if err != nil {
		tflog.Error(ctx, "set old \"image_ids\" error: "+err.Error())
	}

	oldResellerID, _ := d.GetChange(ResellerIDField)
	d.SetId(strconv.Itoa(oldResellerID.(int)))
}

func rollbackResellerImagesV2Data(ctx context.Context, d *schema.ResourceData) {
	resellerImagesOptions, _ := d.GetChange(ResellerImagesOptionsField)
	err := d.Set(ResellerImagesOptionsField, resellerImagesOptions)
	if err != nil {
		tflog.Error(ctx, "set old \"image_ids\" error: "+err.Error())
	}

	oldEntityID, _ := d.GetChange(EntityIDField)
	d.SetId(strconv.Itoa(oldEntityID.(int)))

	oldEntityType, _ := d.GetChange(EntityTypeField)
	d.SetId(oldEntityType.(string))
}

func prepareResellerImagesOptions(d *schema.ResourceData, riList []edgecloudV2.ResellerImage) *schema.Set {
	riOptions := d.Get(ResellerImagesOptionsField).(*schema.Set)

	for _, ri := range riList {
		riOption := make(map[string]interface{})

		if ri.ImageIDs != nil {
			imageIDs := make([]interface{}, 0, len(*ri.ImageIDs))

			for _, imageID := range *ri.ImageIDs {
				imageIDs = append(imageIDs, imageID)
			}

			riOption[ImageIDsField] = schema.NewSet(schema.HashString, imageIDs)
		}

		riOption[RegionIDField] = ri.RegionID
		riOption[CreatedAtField] = ri.CreatedAt
		riOption[UpdatedAtField] = ri.UpdatedAt

		riOptions.Add(riOption)
	}

	return riOptions
}

func prepareResellerImagesV2Options(d *schema.ResourceData, riList []edgecloudV2.ResellerImageV2) *schema.Set {
	riOptions := d.Get(ResellerImagesOptionsField).(*schema.Set)

	for _, ri := range riList {
		riOption := make(map[string]interface{})

		if ri.ImageIDs != nil {
			imageIDs := make([]interface{}, 0, len(*ri.ImageIDs))

			for _, imageID := range *ri.ImageIDs {
				imageIDs = append(imageIDs, imageID)
			}

			riOption[ImageIDsField] = schema.NewSet(schema.HashString, imageIDs)
		}

		riOption[RegionIDField] = ri.RegionID
		riOption[CreatedAtField] = ri.CreatedAt
		riOption[UpdatedAtField] = ri.UpdatedAt

		riOptions.Add(riOption)
	}

	return riOptions
}
