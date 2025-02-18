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
	oldImageIDs, _ := d.GetChange(ImageIDsField)
	err := d.Set(ImageIDsField, oldImageIDs)
	if err != nil {
		tflog.Error(ctx, "set old \"image_ids\" error: "+err.Error())
	}

	oldResellerID, _ := d.GetChange(ResellerIDField)
	d.SetId(strconv.Itoa(oldResellerID.(int)))

	oldRegionID, _ := d.GetChange(RegionIDField)
	err = d.Set(RegionIDField, oldRegionID)
	if err != nil {
		tflog.Error(ctx, "set old \"region_id\" error: "+err.Error())
	}
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
