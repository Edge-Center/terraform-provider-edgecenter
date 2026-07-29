package dns

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	DNSSecondaryZonesDataSource  = "edgecenter_dns_secondary_zones"
	DNSSecondaryZonesSchemaZones = "zones"
)

func dataSourceDNSSecondaryZones() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			DNSSecondaryZonesSchemaZones: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DNSSecondaryZoneSchemaName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the secondary zone",
						},
						DNSSecondaryZoneSchemaMaster: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP address of the primary DNS server",
						},
						DNSSecondaryZoneSchemaTSIGName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "TSIG key name",
						},
						DNSSecondaryZoneSchemaZoneID: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Unique identifier of the secondary zone",
						},
						DNSSecondaryZoneSchemaUpdatedAt: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp of the last update",
						},
					},
				},
				Description: "List of secondary zones",
			},
		},
		ReadContext: checkDNSDependency(dataSourceDNSSecondaryZonesRead),
		Description: "Get list of DNS secondary zones",
	}
}

// dataSourceDNSSecondaryZonesRead reads all secondary zones.
func dataSourceDNSSecondaryZonesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DNS Secondary Zones Data Source reading")
	defer log.Println("[DEBUG] Finish DNS Secondary Zones Data Source reading")

	config := m.(*edgecenter.Config)
	client := config.DNSClient

	zones, err := client.SecondaryZones(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("get secondary zones: %w", err))
	}

	// convert to Terraform format
	zoneList := make([]map[string]interface{}, len(zones))
	for i, zone := range zones {
		// convert Timestamp to string
		var updatedAtStr string
		if zone.UpdatedAt != 0 {
			t := time.Unix(0, int64(zone.UpdatedAt))
			updatedAtStr = t.Format(time.RFC3339)
		}

		zoneMap := map[string]interface{}{
			DNSSecondaryZoneSchemaName:      zone.Name,
			DNSSecondaryZoneSchemaZoneID:    zone.ID,
			DNSSecondaryZoneSchemaUpdatedAt: updatedAtStr,
		}

		if zone.TSIG != nil {
			zoneMap[DNSSecondaryZoneSchemaMaster] = zone.TSIG.Master
			zoneMap[DNSSecondaryZoneSchemaTSIGName] = zone.TSIG.Name
			// TSIG Key is hidden fir security reasons
		}

		zoneList[i] = zoneMap
	}

	if err := d.Set(DNSSecondaryZonesSchemaZones, zoneList); err != nil {
		return diag.FromErr(fmt.Errorf("set zones: %w", err))
	}

	d.SetId("secondary_zones")

	return nil
}
