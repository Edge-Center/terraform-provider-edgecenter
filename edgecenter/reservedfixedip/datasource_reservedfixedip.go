package reservedfixedip

import (
	"context"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceReservedFixedIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterReservedFixedIPRead,
		Description: `A reserved fixed IP is an IP address within a specific network that is reserved for a particular
purpose. Reserved fixed IPs are typically not automatically assigned to instances but are instead set aside for specific
needs or configurations`,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the project",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the region",
			},
			"port_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the port",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the region",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the reserved fixed IP",
			},
			"fixed_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the reserved fixed IP",
			},
			"is_vip": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the reserved fixed IP is a VIP",
			},
			"is_external": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the reserved fixed IP belongs to a public network",
			},
			"network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network that the port is attached to",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the underlying port",
			},
			"subnet_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the subnet that owns the IP address",
			},
			"allowed_address_pairs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A group of subnet masks and/or IP addresses that share the current IP as a VIP",
				Elem:        &schema.Resource{},
			},
			"network": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The details of the network",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"reservation": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The status of the reserved fixed IP with the type of the resource and the ID it is attached to",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceEdgeCenterReservedFixedIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundFixedReservedIP *edgecloud.ReservedFixedIP

	if portId, ok := d.GetOk("port_id"); ok {
		fixedReservedIP, _, err := client.ReservedFixedIP.Get(ctx, portId.(string))
		if err != nil {
			diag.FromErr(err)
		}
		foundFixedReservedIP = fixedReservedIP
	} else {
		return diag.Errorf("Error: specify  port_id  to lookup the reservedfixedip")

	}

	d.SetId(foundFixedReservedIP.PortID)

	d.Set("region", foundFixedReservedIP.Region)
	d.Set("name", foundFixedReservedIP.Name)
	d.Set("region_id", foundFixedReservedIP.RegionID)
	d.Set("fixed_ip_address", foundFixedReservedIP.FixedIPAddress)
	d.Set("is_vip", foundFixedReservedIP.IsVIP)
	d.Set("is_external", foundFixedReservedIP.IsExternal)
	d.Set("project_id", foundFixedReservedIP.ProjectID)
	d.Set("network_id", foundFixedReservedIP.NetworkID)
	d.Set("status", foundFixedReservedIP.Status)
	d.Set("subnet_id", foundFixedReservedIP.SubnetID)

	if err := setNetwork(d, foundFixedReservedIP); err != nil {
		return diag.FromErr(err)
	}
	if err := setReservation(d, foundFixedReservedIP); err != nil {
		return diag.FromErr(err)
	}
	if len(foundFixedReservedIP.AllowedAddressPairs) > 0 {
		allowedAddressPairs := make([]map[string]interface{}, 0, len(foundFixedReservedIP.AllowedAddressPairs))
		for _, allowedAddressPairItem := range foundFixedReservedIP.AllowedAddressPairs {
			allowedAddressPairs = append(allowedAddressPairs, map[string]interface{}{
				"ip_address":  allowedAddressPairItem.IPAddress,
				"mac_address": allowedAddressPairItem.MacAddress,
			})
		}
		if err := d.Set("allowed_address_pairs", allowedAddressPairs); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func setNetwork(d *schema.ResourceData, reservedFixedIP *edgecloud.ReservedFixedIP) error {
	network := map[string]string{
		"id":   reservedFixedIP.Network.ID,
		"name": reservedFixedIP.Name,
	}
	return d.Set("network", network)
}

func setReservation(d *schema.ResourceData, reservedFixedIP *edgecloud.ReservedFixedIP) error {
	reservation := map[string]string{
		"status": reservedFixedIP.Reservation.Status,
	}
	if reservedFixedIP.Reservation.ResourceID != "" {
		reservation["resource_id"] = reservedFixedIP.Reservation.ResourceID
	}
	if reservedFixedIP.Reservation.ResourceType != "" {
		reservation["resource_type"] = reservedFixedIP.Reservation.ResourceType
	}
	return d.Set("reservation", reservation)
}
