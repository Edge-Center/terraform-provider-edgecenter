package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/Edge-Center/edgecentercdn-go/shielding"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataShieldingLocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataShieldingLocationRead,
		Description: "Represent shielding locations.",
		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:        schema.TypeString,
				Description: "Datacenter of shielding location.",
				Required:    true,
			},
		},
	}
}

func getLocationByDC(arr []shielding.ShieldingLocations, datacenter string) (int, error) {
	for _, el := range arr {
		if el.Datacenter == datacenter {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("shielding location for datacenter %s not found", datacenter)
}

func dataShieldingLocationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start reading shielding locations.")

	datacenter := d.Get("datacenter").(string)
	config := m.(*Config)
	client := config.CDNClient

	result, err := client.Shielding().GetShieldingLocations(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[DEBUG] Shielding locations: %v", *result)
	locationID, err := getLocationByDC(*result, datacenter)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(locationID))
	err = d.Set("datacenter", datacenter)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish reading shielding locations")
	return nil
}
