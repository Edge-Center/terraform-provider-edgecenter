package floatingip

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterFloatingIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterFloatingIPRead,
		Description: `A floating IP is a static IP address that can be associated with one of your instances or loadbalancers, 
allowing it to have a static public IP address. The floating IP can be re-associated to any other instance in the same datacenter.`,

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
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "floating IP uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "floating_ip_address"},
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "floating IP address assigned to the resource, must be a valid IP address",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
				ExactlyOneOf: []string{"id", "floating_ip_address"},
			},
			// computed attributes
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "current status ('DOWN' or 'ACTIVE') of the floating IP resource",
			},
			"port_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "network port uuid that the floating IP is associated with",
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
			"fixed_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "fixed IP address",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "name of the region",
			},
			"instance": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "instance that the floating IP is attached to",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"loadbalancer": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "load balancer that the floating IP is attached to",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "metadata in detailed format",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceEdgeCenterFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundFloatingIP *edgecloud.FloatingIP

	if id, ok := d.GetOk("id"); ok {
		floatingIP, err := util.FloatingIPDetailedByID(ctx, client, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundFloatingIP = floatingIP
	} else if floatingIPAddress, ok := d.GetOk("floating_ip_address"); ok {
		floatingIP, err := util.FloatingIPDetailedByIPAddress(ctx, client, floatingIPAddress.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundFloatingIP = floatingIP
	} else {
		return diag.Errorf("Error: specify either a floating_ip_address or id to lookup the floating ip")
	}

	d.SetId(foundFloatingIP.ID)
	d.Set("floating_ip_address", foundFloatingIP.FloatingIPAddress)
	d.Set("status", foundFloatingIP.Status)
	d.Set("port_id", foundFloatingIP.PortID)

	d.Set("router_id", foundFloatingIP.RouterID)
	d.Set("subnet_id", foundFloatingIP.SubnetID)
	d.Set("fixed_ip_address", foundFloatingIP.FixedIPAddress.String())
	d.Set("region", foundFloatingIP.Region)

	if len(foundFloatingIP.Metadata) > 0 {
		metadata := make([]map[string]interface{}, 0, len(foundFloatingIP.Metadata))
		for _, metadataItem := range foundFloatingIP.Metadata {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
		d.Set("metadata", metadata)
	}

	if foundFloatingIP.Instance.ID != "" {
		instance := map[string]string{
			"instance_id":   foundFloatingIP.Instance.ID,
			"instance_name": foundFloatingIP.Instance.Name,
			"status":        foundFloatingIP.Instance.Status,
			"vm_state":      foundFloatingIP.Instance.VMState,
		}
		d.Set("instance", instance)
	}

	if foundFloatingIP.Loadbalancer.ID != "" {
		loadbalancer := map[string]string{
			"id":                  foundFloatingIP.Loadbalancer.ID,
			"provisioning_status": string(foundFloatingIP.Loadbalancer.ProvisioningStatus),
			"operating_status":    string(foundFloatingIP.Loadbalancer.OperatingStatus),
			"name":                foundFloatingIP.Loadbalancer.Name,
			"vip_address":         foundFloatingIP.Loadbalancer.VipAddress.String(),
			"vip_port_id":         foundFloatingIP.Loadbalancer.VipPortID,
			"vip_network_id":      foundFloatingIP.Loadbalancer.VipNetworkID,
		}
		d.Set("loadbalancer", loadbalancer)
	}

	return nil
}
