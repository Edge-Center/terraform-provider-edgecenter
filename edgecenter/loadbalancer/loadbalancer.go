package loadbalancer

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func loadbalancerSchema() map[string]*schema.Schema {
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
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "name of the load balancer",
		},
		"flavor_name": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "flavor name of the load balancer",
		},
		"vip_port_id": {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{"vip_network_id"},
			Description:   "ID of the existing reserved fixed IP port for the load balancer",
		},
		"vip_network_id": {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{"vip_port_id"},
			Description:   "ID of the Network. шf not specified, the default external network will be used",
		},
		"vip_subnet_id": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ForceNew:     true,
			RequiredWith: []string{"vip_network_id"},
			Description:  "ID of the subnet. if not specified, any subnet from vip_network_id will be selected",
		},
		"metadata": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "map containing metadata, for example tags.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"floating_ip_source": {
			Type:         schema.TypeString,
			Optional:     true,
			ForceNew:     true,
			Description:  "floating IP type: 'existing' or 'new'",
			RequiredWith: []string{"vip_network_id"},
		},
		"floating_ip": {
			Type:         schema.TypeString,
			ValidateFunc: validation.IsUUID,
			Optional:     true,
			Computed:     true,
			Description:  "floating IP for this subnet attachment",
		},
		// computed attributes
		"region": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the region",
		},
		"vip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "IP address of the load balancer",
		},
		"provisioning_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "lifecycle status of the load balancer",
		},
		"operating_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "operating status of the load balancer",
		},
		"vrrp_ips": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "list of VRRP IP addresses",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"flavor": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "information about the flavor",
		},
	}
}

/*
   "listeners": [],
   "floating_ips": [],
*/
