package lbpool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterLbPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterLbPoolRead,
		Description: `A pool is a list of virtual machines to which the listener will redirect incoming traffic`,

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
				Description:  "lb pool uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Description: `lb pool name. this parameter is not unique, if there is more than one lb pool with the same name, 
then the first one will be used. it is recommended to use "id"`,
				ExactlyOneOf: []string{"id", "name"},
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the load balancer",
			},
			// computed attributes
			"listener_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the load balancer listener",
			},
			"lb_algorithm": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "algorithm of the load balancer",
			},
			"provisioning_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "lifecycle status of the pool",
			},
			"session_persistence": {
				Type:     schema.TypeList,
				Computed: true,
				Description: `configuration that enables the load balancer to bind a user's session to a specific backend member. 
this ensures that all requests from the user during the session are sent to the same member.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cookie_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"persistence_granularity": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"persistence_timeout": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"timeout_member_connect": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "timeout for the backend member connection (in milliseconds)",
			},
			"timeout_member_data": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "timeout for the backend member inactivity (in milliseconds)",
			},
			"timeout_client_data": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "timeout for the frontend client inactivity (in milliseconds)",
			},
			"healthmonitor": {
				Type:     schema.TypeList,
				Computed: true,
				Description: `configuration for health checks to test the health and state of the backend members. 
it determines how the load balancer identifies whether the backend members are healthy or unhealthy`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"delay": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"timeout": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_retries": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_retries_down": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"url_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"expected_codes": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"http_method": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"operating_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "operating status of the pool",
			},
			"protocol": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "protocol of the load balancer",
			},
			"member": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "members of the Pool",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"weight": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"protocol_port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"operating_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"admin_state_up": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceEdgeCenterLbPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	loadbalancerID := d.Get("loadbalancer_id").(string)

	var foundPool *edgecloud.Pool

	if id, ok := d.GetOk("id"); ok {
		pool, _, err := client.Loadbalancers.PoolGet(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundPool = pool
	} else if poolName, ok := d.GetOk("name"); ok {
		pool, err := util.LBPoolGetByName(ctx, client, poolName.(string), loadbalancerID)
		if err != nil {
			return diag.FromErr(err)
		}

		foundPool = pool
	} else {
		return diag.Errorf("Error: specify either id or a name to lookup the lb pool")
	}

	d.SetId(foundPool.ID)
	d.Set("name", foundPool.Name)
	d.Set("lb_algorithm", foundPool.LoadbalancerAlgorithm)
	d.Set("protocol", foundPool.Protocol)
	d.Set("provisioning_status", foundPool.ProvisioningStatus)
	d.Set("operating_status", foundPool.OperatingStatus)
	d.Set("listener_id", foundPool.Listeners[0].ID)
	d.Set("timeout_member_connect", foundPool.TimeoutMemberConnect)
	d.Set("timeout_member_data", foundPool.TimeoutMemberData)
	d.Set("timeout_client_data", foundPool.TimeoutClientData)

	if err := setHealthMonitor(ctx, d, foundPool); err != nil {
		return diag.FromErr(err)
	}

	if err := setSessionPersistence(ctx, d, foundPool); err != nil {
		return diag.FromErr(err)
	}

	if err := setMembers(ctx, d, foundPool); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
