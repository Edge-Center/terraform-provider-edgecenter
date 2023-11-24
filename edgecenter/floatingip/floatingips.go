package floatingip

import (
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func floatingIPSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the project",
		},
		"region_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the region",
		},
		"fixed_ip_address": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			Description: `in case the port has multiple IPs, a specific address can be selected using this field. 
if unspecified, the first IP in the list of the port list is used. must be a valid IP address`,
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				ip := net.ParseIP(v)
				if ip != nil {
					return diag.Diagnostics{}
				}

				return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
			},
			RequiredWith: []string{"port_id"},
		},
		"port_id": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			Description:  "network port uuid, if provided, the floating IP will be immediately attached to the specified port",
			RequiredWith: []string{"fixed_ip_address"},
		},
		"metadata": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "map containing metadata, for example tags.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		// computed attributes
		"floating_ip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "floating IP address assigned to the resource",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "current status ('DOWN' or 'ACTIVE') of the floating IP resource",
		},
		"router_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of the router",
		},
		"subnet_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of the subnet",
		},
		"region": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the region",
		},
	}
}
