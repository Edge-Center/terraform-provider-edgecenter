package reservedfixedip

import (
	"fmt"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net"
)

func reservedFixedIPSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project_id": {
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
			ExactlyOneOf: []string{"project_id", "project_name"},
		},
		"project_name": {
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
			ExactlyOneOf: []string{"project_id", "project_name"},
		},
		"region_id": {
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
			ExactlyOneOf: []string{"region_id", "region_name"},
		},
		"region_name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
			ExactlyOneOf: []string{"region_id", "region_name"},
		},
		"type": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: fmt.Sprintf("The type of reserved fixed IP. Valid values are '%s', '%s', '%s', and '%s'", edgecloud.ReservedFixedIPTypeExternal, edgecloud.ReservedFixedIPTypeSubnet, edgecloud.ReservedFixedIPTypeAnySubnet, edgecloud.ReservedFixedIPTypeIPAddress),
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				switch edgecloud.ReservedFixedIPType(v) {
				case edgecloud.ReservedFixedIPTypeExternal, edgecloud.ReservedFixedIPTypeSubnet, edgecloud.ReservedFixedIPTypeAnySubnet, edgecloud.ReservedFixedIPTypeIPAddress:
					return diag.Diagnostics{}
				}
				return diag.Errorf("wrong type %s, available values is '%s', '%s', '%s', '%s'", v, edgecloud.ReservedFixedIPTypeExternal, edgecloud.ReservedFixedIPTypeSubnet, edgecloud.ReservedFixedIPTypeAnySubnet, edgecloud.ReservedFixedIPTypeIPAddress)
			},
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The current status of the reserved fixed IP.",
		},
		"fixed_ip_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "The IP address that is associated with the reserved IP.",
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				ip := net.ParseIP(v)
				if ip != nil {
					return diag.Diagnostics{}
				}

				return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
			},
		},
		"subnet_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "ID of the subnet from which the fixed IP should be reserved.",
		},
		"network_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "ID of the network to which the reserved fixed IP is associated.",
		},
		"network_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the network",
		},
		"is_vip": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Flag to determine if the reserved fixed IP should be treated as a Virtual IP (VIP).",
		},
		"port_id": {
			Type:        schema.TypeString,
			Description: "ID of the port_id underlying the reserved fixed IP.",
			Computed:    true,
		},
		"allowed_address_pairs": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Group of IP addresses that share the current IP as VIP.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ip_address": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"mac_address": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"last_updated": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The timestamp of the last update (use with update context).",
		},
		"reservation": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "The status of the reserved fixed IP with the type of the resource and the ID it is attached to",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"instance_ports_that_share_vip": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "instance ports that share a VIP",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
}
