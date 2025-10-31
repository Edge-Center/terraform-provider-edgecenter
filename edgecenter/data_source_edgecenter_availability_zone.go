package edgecenter

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAvailabilityZone() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAvailabilityZonesRead,
		Description: "Represent Availability Zones",
		Schema: map[string]*schema.Schema{
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The ID of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"availability_zones": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of availability zones in the region.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceAvailabilityZonesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start Availability Zones reading")

	clientConf := CloudClientConf{
		DoNotUseProjectID: true,
	}
	clientV2, err := InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	az, _, err := clientV2.AvailabilityZones.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(az.RegionID))
	if err := d.Set("availability_zones", az.AvailabilityZones); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish Availability Zones reading")

	return nil
}
