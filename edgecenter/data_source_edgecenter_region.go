package edgecenter

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRegion() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRegionRead,
		Description: "Represent region data",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Displayed region name",
				Required:    true,
			},
		},
	}
}

func dataSourceRegionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Region reading")

	name := d.Get("name").(string)

	clientConf := CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
	clientV2, err := InitCloudClient(ctx, d, m, &clientConf)
	if err != nil {
		return diag.FromErr(err)
	}

	regionID, err := GetRegionV2(ctx, clientV2, 0, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(regionID))
	d.Set("name", name)

	log.Println("[DEBUG] Finish Region reading")

	return nil
}
