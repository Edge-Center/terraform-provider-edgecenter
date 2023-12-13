package lbpool

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func lbpoolSchema() map[string]*schema.Schema {
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
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: `lb pool name`,
		},
		"lb_algorithm": {
			Type:     schema.TypeString,
			Required: true,
			Description: fmt.Sprintf(
				"algorithm of the load balancer. available values are '%s', '%s', '%s', '%s'",
				edgecloud.LoadbalancerAlgorithmRoundRobin, edgecloud.LoadbalancerAlgorithmLeastConnections,
				edgecloud.LoadbalancerAlgorithmSourceIP, edgecloud.LoadbalancerAlgorithmSourceIPPort,
			),
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				switch edgecloud.LoadbalancerAlgorithm(v) {
				case edgecloud.LoadbalancerAlgorithmRoundRobin, edgecloud.LoadbalancerAlgorithmLeastConnections, edgecloud.LoadbalancerAlgorithmSourceIP, edgecloud.LoadbalancerAlgorithmSourceIPPort:
					return diag.Diagnostics{}
				}

				return diag.Errorf(
					"wrong type %s, available values are '%s', '%s', '%s', '%s'", v,
					edgecloud.LoadbalancerAlgorithmRoundRobin, edgecloud.LoadbalancerAlgorithmLeastConnections,
					edgecloud.LoadbalancerAlgorithmSourceIP, edgecloud.LoadbalancerAlgorithmSourceIPPort,
				)
			},
		},
		"protocol": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			Description: fmt.Sprintf(
				"available values are '%s', '%s', '%s', '%s' and '%s'",
				edgecloud.LBPoolProtocolHTTP, edgecloud.LBPoolProtocolHTTPS, edgecloud.LBPoolProtocolTCP,
				edgecloud.LBPoolProtocolUDP, edgecloud.LBPoolProtocolProxy,
			),
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				switch edgecloud.LoadbalancerPoolProtocol(v) {
				case edgecloud.LBPoolProtocolHTTP, edgecloud.LBPoolProtocolHTTPS, edgecloud.LBPoolProtocolTCP,
					edgecloud.LBPoolProtocolUDP, edgecloud.LBPoolProtocolProxy:
					return diag.Diagnostics{}
				default:
					return diag.Errorf(
						"wrong protocol %s, available values are '%s', '%s', '%s', '%s', '%s'", v,
						edgecloud.LBPoolProtocolHTTP, edgecloud.LBPoolProtocolHTTPS, edgecloud.LBPoolProtocolTCP,
						edgecloud.LBPoolProtocolUDP, edgecloud.LBPoolProtocolProxy,
					)
				}
			},
		},
		"loadbalancer_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the load balancer",
		},
		"listener_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the load balancer listener",
		},
		"session_persistence": {
			Type:     schema.TypeList,
			Optional: true,
			Computed: true,
			MaxItems: 1,
			Description: `configuration that enables the load balancer to bind a user's session to a specific backend member. 
this ensures that all requests from the user during the session are sent to the same member.`,
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
		"timeout_member_connect": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5000, //nolint: gomnd
			Description: "timeout for the backend member connection (in milliseconds)",
		},
		"timeout_member_data": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5000, //nolint: gomnd
			Description: "timeout for the backend member inactivity (in milliseconds)",
		},
		"timeout_client_data": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5000, //nolint: gomnd
			Description: "timeout for the frontend client inactivity (in milliseconds)",
		},
		"healthmonitor": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Description: `configuration for health checks to test the health and state of the backend members. 
it determines how the load balancer identifies whether the backend members are healthy or unhealthy.`,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Type:     schema.TypeString,
						Required: true,
						Description: fmt.Sprintf(
							"available values are '%s', '%s', '%s', '%s', '%s', '%s",
							edgecloud.HealthMonitorTypeHTTP, edgecloud.HealthMonitorTypeHTTPS,
							edgecloud.HealthMonitorTypePING, edgecloud.HealthMonitorTypeTCP,
							edgecloud.HealthMonitorTypeTLSHello, edgecloud.HealthMonitorTypeUDPConnect),
						ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
							v := val.(string)
							switch edgecloud.HealthMonitorType(v) {
							case edgecloud.HealthMonitorTypeHTTP, edgecloud.HealthMonitorTypeHTTPS,
								edgecloud.HealthMonitorTypePING, edgecloud.HealthMonitorTypeTCP,
								edgecloud.HealthMonitorTypeTLSHello, edgecloud.HealthMonitorTypeUDPConnect:
								return diag.Diagnostics{}
							}

							return diag.Errorf(
								"wrong type %s, available values is '%s', '%s', '%s', '%s', '%s', '%s", v,
								edgecloud.HealthMonitorTypeHTTP, edgecloud.HealthMonitorTypeHTTPS,
								edgecloud.HealthMonitorTypePING, edgecloud.HealthMonitorTypeTCP,
								edgecloud.HealthMonitorTypeTLSHello, edgecloud.HealthMonitorTypeUDPConnect,
							)
						},
					},
					"timeout": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     5, //nolint: gomnd
						Description: "Response time (in sec)",
					},
					"delay": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     60, //nolint: gomnd
						Description: "check interval (in sec)",
					},
					"max_retries": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     10, //nolint: gomnd
						Description: "healthy thresholds",
					},
					"max_retries_down": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     5, //nolint: gomnd
						Description: "unhealthy thresholds",
					},
					"http_method": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"url_path": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"expected_codes": {
						Type:     schema.TypeString,
						Optional: true,
						Computed: true,
					},
					"id": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		// computed attributes
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "lb pool uuid",
		},
		"provisioning_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "lifecycle status of the pool",
		},
		"operating_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "operating status of the pool",
		},
	}
}
