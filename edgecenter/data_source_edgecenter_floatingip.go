package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/edgecentercloud-go/v2/util"
)

// TODO https://tracker.yandex.ru/CLOUDDEV-152
func dataSourceFloatingIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFloatingIPRead,
		Description: `A floating IP is a static IP address that can be associated with one of your instances or loadbalancers, 
allowing it to have a static public IP address. The floating IP can be re-associated to any other instance in the same datacenter.`,

		Schema: map[string]*schema.Schema{
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
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "floating IP uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "floating_ip_address"},
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
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The floating IP address assigned to the resource. It must be a valid IP address.",
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
			"port_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID (uuid) of the network port that the floating IP is associated with.",
			},
			"instance_id_attached_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the instance, that the floating IP is associated with.",
			},
			"load_balancers_id_attached_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the loadbalancer, that the floating IP associated with",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the floating IP resource. Can be 'DOWN' or 'ACTIVE'.",
			},
			"fixed_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The fixed (reserved) IP address that is associated with the floating IP.",
			},
			"router_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the router that the floating IP is associated with.",
			},
			"metadata_k": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			"metadata_kv": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}.`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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

func dataSourceFloatingIPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP reading")
	var diags diag.Diagnostics

	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}
	clientV2.Region = regionID
	clientV2.Project = projectID

	var foundFloatingIP *edgecloudV2.FloatingIP

	if id, ok := d.GetOk("id"); ok {
		floatingIP, err := util.FloatingIPDetailedByID(ctx, clientV2, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		foundFloatingIP = floatingIP
	} else if floatingIPAddress, ok := d.GetOk("floating_ip_address"); ok {
		floatingIP, err := util.FloatingIPDetailedByIPAddress(ctx, clientV2, floatingIPAddress.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		foundFloatingIP = floatingIP
	} else {
		return diag.Errorf("Error: specify either a floating_ip_address or id to lookup the floating ip")
	}
	d.SetId(foundFloatingIP.ID)

	if foundFloatingIP.FixedIPAddress != nil {
		d.Set("fixed_ip_address", foundFloatingIP.FixedIPAddress.String())
	} else {
		d.Set("fixed_ip_address", "")
	}

	d.Set("project_id", foundFloatingIP.ProjectID)
	d.Set("region_id", foundFloatingIP.RegionID)
	d.Set("status", foundFloatingIP.Status)
	d.Set("port_id", foundFloatingIP.PortID)
	if foundFloatingIP.Instance.ID != "" {
		d.Set("instance_id_attached_to", foundFloatingIP.Instance.ID)
	}
	if foundFloatingIP.Loadbalancer.ID != "" {
		d.Set("load_balancer_id_attached_to", foundFloatingIP.Loadbalancer.ID)
	}
	d.Set("router_id", foundFloatingIP.RouterID)
	d.Set("floating_ip_address", foundFloatingIP.FloatingIPAddress)

	metadataReadOnly := PrepareMetadataReadonly(foundFloatingIP.Metadata)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish FloatingIP reading")

	return diags
}
