package edgecenter

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceLBPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLBPoolRead,
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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the load balancer pool.",
			},
			"lb_algorithm": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Available values is '%s', '%s', '%s', '%s'", types.LoadBalancerAlgorithmRoundRobin, types.LoadBalancerAlgorithmLeastConnections, types.LoadBalancerAlgorithmSourceIP, types.LoadBalancerAlgorithmSourceIPPort),
			},
			"protocol": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Available values is '%s' (currently work, other do not work on ed-8), '%s', '%s', '%s'", types.ProtocolTypeHTTP, types.ProtocolTypeHTTPS, types.ProtocolTypeTCP, types.ProtocolTypeUDP),
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid for the load balancer.",
			},
			"listener_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid for the load balancer listener.",
			},
			"health_monitor": {
				Type:     schema.TypeList,
				Computed: true,
				Description: `Configuration for health checks to test the health and state of the backend members. 
It determines how the load balancer identifies whether the backend members are healthy or unhealthy.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("Available values is '%s', '%s', '%s', '%s', '%s', '%s", types.HealthMonitorTypeHTTP, types.HealthMonitorTypeHTTPS, types.HealthMonitorTypePING, types.HealthMonitorTypeTCP, types.HealthMonitorTypeTLSHello, types.HealthMonitorTypeUDPConnect),
						},
						"delay": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_retries": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"timeout": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"max_retries_down": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"http_method": {
							Type:     schema.TypeString,
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
					},
				},
			},
			"session_persistence": {
				Type:     schema.TypeList,
				Computed: true,
				Description: `Configuration that enables the load balancer to bind a user's session to a specific backend member. 
This ensures that all requests from the user during the session are sent to the same member.`,
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
		},
	}
}

func dataSourceLBPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	clientV2.Region = d.Get("region_id").(int)
	clientV2.Project = d.Get("project_id").(int)

	var opts edgecloudV2.PoolListOptions
	name := d.Get("name").(string)
	lbID := d.Get("loadbalancer_id").(string)
	if lbID != "" {
		opts.LoadbalancerID = lbID
	}
	lID := d.Get("listener_id").(string)
	if lbID != "" {
		opts.ListenerID = lID
	}

	pools, _, err := clientV2.Loadbalancers.PoolList(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var lb edgecloudV2.Pool
	for _, p := range pools {
		if p.Name == name {
			lb = p
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("lb listener with name %s not found", name)
	}

	d.SetId(lb.ID)
	d.Set("name", lb.Name)
	d.Set("lb_algorithm", lb.LoadbalancerAlgorithm)
	d.Set("protocol", lb.Protocol)

	if len(lb.Loadbalancers) > 0 {
		d.Set("loadbalancer_id", lb.Loadbalancers[0].ID)
	}

	if len(lb.Listeners) > 0 {
		d.Set("listener_id", lb.Listeners[0].ID)
	}

	if lb.HealthMonitor != nil {
		healthMonitor := map[string]interface{}{
			"id":               lb.HealthMonitor.ID,
			"type":             lb.HealthMonitor.Type,
			"delay":            lb.HealthMonitor.Delay,
			"timeout":          lb.HealthMonitor.Timeout,
			"max_retries":      lb.HealthMonitor.MaxRetries,
			"max_retries_down": lb.HealthMonitor.MaxRetriesDown,
			"url_path":         lb.HealthMonitor.URLPath,
			"expected_codes":   lb.HealthMonitor.ExpectedCodes,
		}
		if lb.HealthMonitor.HTTPMethod != nil {
			healthMonitor["http_method"] = lb.HealthMonitor.HTTPMethod
		}

		if err := d.Set("health_monitor", []interface{}{healthMonitor}); err != nil {
			return diag.FromErr(err)
		}
	}

	if lb.SessionPersistence != nil {
		sessionPersistence := map[string]interface{}{
			"type":                    lb.SessionPersistence.Type,
			"cookie_name":             lb.SessionPersistence.CookieName,
			"persistence_granularity": lb.SessionPersistence.PersistenceGranularity,
			"persistence_timeout":     lb.SessionPersistence.PersistenceTimeout,
		}

		if err := d.Set("session_persistence", []interface{}{sessionPersistence}); err != nil {
			return diag.FromErr(err)
		}
	}

	d.Set("project_id", d.Get("project_id").(int))
	d.Set("region_id", d.Get("region_id").(int))

	log.Println("[DEBUG] Finish LBPool reading")

	return diags
}
