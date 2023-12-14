package lbmember

import (
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	minWeight = 0
	maxWeight = 256
)

func lbmemberSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
		"pool_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the load balancer pool",
		},
		"address": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "IP address of the load balancer pool member",
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				ip := net.ParseIP(v)
				if ip != nil {
					return diag.Diagnostics{}
				}

				return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
			},
		},
		"protocol_port": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "IP port on which the member listens for requests",
		},
		"subnet_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "uuid of the subnet in which the pool member is located.",
		},
		"instance_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "uuid of the instance (amphora) associated with the pool member.",
		},
		"weight": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     1,
			Description: "weight value between 0 and 256, determining the distribution of requests among the members of the pool. defaults to 1",
			ValidateDiagFunc: func(val interface{}, path cty.Path) diag.Diagnostics {
				v := val.(int)
				if v >= minWeight && v <= maxWeight {
					return nil
				}
				return diag.Errorf("valid values: %d to %d got: %d", minWeight, maxWeight, v)
			},
		},
		"admin_state_up": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "true if enabled. Defaults to true",
		},
		// computed attributes
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of the member must be provided if the existing member is being updated",
		},
		"operating_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "operating status of the pool",
		},
	}
}
