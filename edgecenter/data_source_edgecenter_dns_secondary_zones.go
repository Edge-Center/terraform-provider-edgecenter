package edgecenter

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDNSSecondaryZones() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zones": {
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
