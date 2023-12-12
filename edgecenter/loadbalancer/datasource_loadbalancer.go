package loadbalancer

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterLoadbalancer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterLoadbalancerRead,
		Description: `A loadbalancer is a software service that distributes incoming network traffic 
(e.g., web traffic, application requests) across multiple servers or resources.`,

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
				Description:  "loadbalancer uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Description: `loadbalancer name. this parameter is not unique, if there is more than one loadbalancer with the same name, 
then the first one will be used. it is recommended to use "id"`,
				ExactlyOneOf: []string{"id", "name"},
			},
			// computed attributes
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "name of the region",
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
			"vip_port_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP port of the load balancer",
			},
			"vip_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the network that the subnet belongs to. the port will be plugged in this network",
			},
			"vrrp_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of VRRP IP addresses",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"vip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "loadbalancer IP address",
			},
			"flavor": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "information about the flavor",
			},
			"floating_ip": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "information about the assigned floating IP",
			},
			"metadata_detailed": {
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

func dataSourceEdgeCenterLoadbalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundLoadbalancer *edgecloud.Loadbalancer

	if id, ok := d.GetOk("id"); ok {
		loadbalancer, _, err := client.Loadbalancers.Get(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundLoadbalancer = loadbalancer
	} else if loadbalancerName, ok := d.GetOk("name"); ok {
		loadbalancer, err := util.LoadbalancerGetByName(ctx, client, loadbalancerName.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundLoadbalancer = loadbalancer
	} else {
		return diag.Errorf("Error: specify either id or a name to lookup the loadbalancer")
	}

	d.SetId(foundLoadbalancer.ID)
	d.Set("name", foundLoadbalancer.Name)

	d.Set("region", foundLoadbalancer.Region)
	d.Set("provisioning_status", foundLoadbalancer.ProvisioningStatus)
	d.Set("operating_status", foundLoadbalancer.OperatingStatus)
	d.Set("vip_port_id", foundLoadbalancer.VipPortID)
	d.Set("vip_network_id", foundLoadbalancer.VipNetworkID)
	d.Set("vip_address", foundLoadbalancer.VipAddress.String())

	if err := setMetadataDetailed(ctx, d, foundLoadbalancer); err != nil {
		return diag.FromErr(err)
	}

	if err := setFlavor(ctx, d, foundLoadbalancer); err != nil {
		return diag.FromErr(err)
	}

	if err := setVRRPIPs(ctx, d, foundLoadbalancer); err != nil {
		return diag.FromErr(err)
	}

	if err := setFloatingIP(ctx, d, foundLoadbalancer); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
