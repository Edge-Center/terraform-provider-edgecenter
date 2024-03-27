package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	LBPoolsPoint         = "lbpools"
	LBPoolsCreateTimeout = 2400 * time.Second
	LBPoolsUpdateTimeout = 2400 * time.Second
	LBPoolsDeleteTimeout = 2400 * time.Second
)

func resourceLBPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLBPoolCreate,
		ReadContext:   resourceLBPoolRead,
		UpdateContext: resourceLBPoolUpdate,
		DeleteContext: resourceLBPoolDelete,
		Description:   "Represent load balancer listener pool. A pool is a list of virtual machines to which the listener will redirect incoming traffic",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, lbPoolID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(lbPoolID)

				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the load balancer listener pool.",
			},
			"lb_algorithm": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("Available values is `%s`, `%s`, `%s`", edgecloudV2.LoadbalancerAlgorithmRoundRobin, edgecloudV2.LoadbalancerAlgorithmLeastConnections, edgecloudV2.LoadbalancerAlgorithmSourceIP),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch edgecloudV2.LoadbalancerAlgorithm(v) {
					case edgecloudV2.LoadbalancerAlgorithmRoundRobin, edgecloudV2.LoadbalancerAlgorithmLeastConnections, edgecloudV2.LoadbalancerAlgorithmSourceIP:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`, `%s`", v, edgecloudV2.LoadbalancerAlgorithmRoundRobin, edgecloudV2.LoadbalancerAlgorithmLeastConnections, edgecloudV2.LoadbalancerAlgorithmSourceIP)
				},
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("Available values is '%s' (currently work, other do not work on ed-8), '%s', '%s', '%s', '%s'", edgecloudV2.LBPoolProtocolHTTP, edgecloudV2.LBPoolProtocolHTTPS, edgecloudV2.LBPoolProtocolTCP, edgecloudV2.LBPoolProtocolUDP, edgecloudV2.LBPoolProtocolProxy),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch edgecloudV2.LoadbalancerPoolProtocol(v) {
					case edgecloudV2.LBPoolProtocolHTTP, edgecloudV2.LBPoolProtocolHTTPS, edgecloudV2.LBPoolProtocolTCP, edgecloudV2.LBPoolProtocolUDP, edgecloudV2.LBPoolProtocolProxy:
						return diag.Diagnostics{}
					case edgecloudV2.LBPoolProtocolTerminatedHTTPS:
					}
					return diag.Errorf("wrong type %s, available values is '%s', '%s', '%s', '%s', '%s'", v, edgecloudV2.LBPoolProtocolHTTP, edgecloudV2.LBPoolProtocolHTTPS, edgecloudV2.LBPoolProtocolTCP, edgecloudV2.LBPoolProtocolUDP, edgecloudV2.LBPoolProtocolProxy)
				},
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The uuid for the load balancer.",
			},
			"listener_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The uuid for the load balancer listener.",
			},
			"health_monitor": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Description: `Configuration for health checks to test the health and state of the backend members. 
It determines how the load balancer identifies whether the backend members are healthy or unhealthy.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available values is '%s', '%s', '%s', '%s', '%s', '%s", edgecloudV2.HealthMonitorTypeHTTP, edgecloudV2.HealthMonitorTypeHTTPS, edgecloudV2.HealthMonitorTypePING, edgecloudV2.HealthMonitorTypeTCP, edgecloudV2.HealthMonitorTypeTLSHello, edgecloudV2.HealthMonitorTypeUDPConnect),
							ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
								v := val.(string)
								switch edgecloudV2.HealthMonitorType(v) {
								case edgecloudV2.HealthMonitorTypeHTTP, edgecloudV2.HealthMonitorTypeHTTPS, edgecloudV2.HealthMonitorTypePING, edgecloudV2.HealthMonitorTypeTCP, edgecloudV2.HealthMonitorTypeTLSHello, edgecloudV2.HealthMonitorTypeUDPConnect:
									return diag.Diagnostics{}
								}
								return diag.Errorf("wrong type %s, available values is '%s', '%s', '%s', '%s', '%s', '%s", v, edgecloudV2.HealthMonitorTypeHTTP, edgecloudV2.HealthMonitorTypeHTTPS, edgecloudV2.HealthMonitorTypePING, edgecloudV2.HealthMonitorTypeTCP, edgecloudV2.HealthMonitorTypeTLSHello, edgecloudV2.HealthMonitorTypeUDPConnect)
							},
						},
						"delay": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"max_retries": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"timeout": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"max_retries_down": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"http_method": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"url_path": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"expected_codes": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"session_persistence": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Description: `Configuration that enables the load balancer to bind a user's session to a specific backend member. 
This ensures that all requests from the user during the session are sent to the same member.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"cookie_name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"persistence_granularity": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"persistence_timeout": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
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
		},
	}
}

func resourceLBPoolCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	healthOpts := extractHealthMonitorMapV2(d)
	sessionOpts := extractSessionPersistenceMapV2(d)
	opts := edgecloudV2.LoadbalancerPoolCreateRequest{
		Name:                  d.Get("name").(string),
		Protocol:              edgecloudV2.LoadbalancerPoolProtocol(d.Get("protocol").(string)),
		LoadbalancerAlgorithm: edgecloudV2.LoadbalancerAlgorithm(d.Get("lb_algorithm").(string)),
		LoadbalancerID:        d.Get("loadbalancer_id").(string),
		ListenerID:            d.Get("listener_id").(string),
		HealthMonitor:         healthOpts,
		SessionPersistence:    sessionOpts,
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Loadbalancers.PoolCreate, &edgecloudV2.PoolCreateRequest{LoadbalancerPoolCreateRequest: opts}, clientV2, LBPoolsCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	lbPoolID := taskResult.Pools[0]

	d.SetId(lbPoolID)
	resourceLBPoolRead(ctx, d, m)

	log.Printf("[DEBUG] Finish LBPool creating (%s)", lbPoolID)

	return diags
}

func resourceLBPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	lb, _, err := clientV2.Loadbalancers.PoolGet(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

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

	fields := []string{"project_id", "region_id"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish LBPool reading")

	return diags
}

func resourceLBPoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool updating")
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	var change bool
	opts := edgecloudV2.PoolUpdateRequest{Name: d.Get("name").(string)}

	if d.HasChange("lb_algorithm") {
		opts.LoadbalancerAlgorithm = edgecloudV2.LoadbalancerAlgorithm(d.Get("lb_algorithm").(string))
		change = true
	}

	if d.HasChange("health_monitor") {
		opts.HealthMonitor = extractHealthMonitorMapV2(d)
		change = true
	}

	if d.HasChange("session_persistence") {
		opts.SessionPersistence = extractSessionPersistenceMapV2(d)
		change = true
	}

	if !change {
		log.Println("[DEBUG] Finish LBPool updating")
		return resourceLBPoolRead(ctx, d, m)
	}

	task, _, err := clientV2.Loadbalancers.PoolUpdate(ctx, d.Id(), &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := task.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, LBPoolsUpdateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish LBPool updating")

	return resourceLBPoolRead(ctx, d, m)
}

func resourceLBPoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	id := d.Id()
	results, resp, err := clientV2.Loadbalancers.PoolDelete(ctx, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of LBPool deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, LBPoolsDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LBPool deleting")

	return diags
}
