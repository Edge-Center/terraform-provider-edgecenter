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
	LoadBalancersPoint        = "loadbalancers"
	LoadBalancerCreateTimeout = 2400 * time.Second
	LoadBalancerDeleteTimeout = 2400 * time.Second
	LoadBalancerUpdateTimeout = 2400 * time.Second
)

func resourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This resource is deprecated and will be removed in the next major version. Use edgecenter_loadbalancerv2 resource instead",
		CreateContext:      resourceLoadBalancerCreate,
		ReadContext:        resourceLoadBalancerRead,
		UpdateContext:      resourceLoadBalancerUpdate,
		DeleteContext:      resourceLoadBalancerDelete,
		Description:        "Represent load balancer",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, lbID, listenerID, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(lbID)

				config := m.(*Config)
				clientV2 := config.CloudClient

				clientV2.Region = regionID
				clientV2.Project = projectID

				listener, _, err := clientV2.Loadbalancers.ListenerGet(ctx, listenerID)
				if err != nil {
					return nil, fmt.Errorf("extracting Listener resource error: %w", err)
				}

				l := extractListenerIntoMapV2(listener)
				if err := d.Set("listener", []interface{}{l}); err != nil {
					return nil, fmt.Errorf("set listener error: %w", err)
				}

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
				Description: "The name of the load balancer.",
			},
			"flavor": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"vip_network_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"vip_subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"vip_address": {
				Type:        schema.TypeString,
				Description: "Load balancer IP address",
				Computed:    true,
			},
			"listener": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"certificate": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"protocol": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available values is '%s' (currently work, other do not work on ed-8), '%s', '%s', '%s'", edgecloudV2.ListenerProtocolTCP, edgecloudV2.ListenerProtocolUDP, edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolHTTPS),
							ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
								v := val.(string)
								switch edgecloudV2.LoadbalancerListenerProtocol(v) {
								case edgecloudV2.ListenerProtocolTCP, edgecloudV2.ListenerProtocolUDP, edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolHTTPS:
									return diag.Diagnostics{}
								case edgecloudV2.ListenerProtocolTerminatedHTTPS:
								}
								return diag.Errorf("wrong protocol %s, available values is 'HTTP', 'HTTPS', 'TCP', 'UDP'", v)
							},
						},
						"certificate_chain": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"protocol_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"private_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"insert_x_forwarded": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"secret_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"sni_secret_id": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
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
			"metadata_map": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
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

func resourceLoadBalancerCreate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("use edgecenter_loadbalancerv2 resource instead"))
}

func resourceLoadBalancerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	lb, _, err := clientV2.Loadbalancers.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("project_id", lb.ProjectID)
	d.Set("region_id", lb.RegionID)
	d.Set("name", lb.Name)
	d.Set("flavor", lb.Flavor.FlavorName)

	if lb.VipAddress != nil {
		d.Set("vip_address", lb.VipAddress.String())
	}

	fields := []string{"vip_network_id", "vip_subnet_id"}
	revertState(d, &fields)

	var ok bool
	currentL := make(map[string]interface{})
	// we need to find correct listener because after upgrade some of them could be nil
	// but still in terraform.state
	cls := d.Get("listener").([]interface{})
	for _, cl := range cls {
		if currentL, ok = cl.(map[string]interface{}); ok {
			break
		}
	}

	for _, l := range lb.Listeners {
		listener, _, err := clientV2.Loadbalancers.ListenerGet(ctx, l.ID)
		if err != nil {
			return diag.FromErr(err)
		}
		port, _ := currentL["protocol_port"].(int)
		if (listener.ProtocolPort == port && listener.Protocol == currentL["protocol"]) || len(cls) == 0 {
			currentL = extractListenerIntoMapV2(listener)
			break
		}
	}
	if err := d.Set("listener", []interface{}{currentL}); err != nil {
		diag.FromErr(err)
	}

	metadataMap, metadataReadOnly := PrepareMetadata(lb.MetadataDetailed)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish LoadBalancer reading")

	return diags
}

func resourceLoadBalancerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer updating")
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	if d.HasChange("name") {
		opts := &edgecloudV2.Name{
			Name: d.Get("name").(string),
		}
		if _, _, err = clientV2.Loadbalancers.Rename(ctx, d.Id(), opts); err != nil {
			return diag.FromErr(err)
		}

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("listener") {
		oldListenerRaw, newListenerRaw := d.GetChange("listener")
		oldListener := oldListenerRaw.([]interface{})[0].(map[string]interface{})
		newListener := newListenerRaw.([]interface{})[0].(map[string]interface{})

		listenerID := oldListener["id"].(string)
		if oldListener["protocol"].(string) != newListener["protocol"].(string) ||
			oldListener["protocol_port"].(int) != newListener["protocol_port"].(int) {
			// if protocol or port changed listener need to be recreated
			// delete at first
			results, _, err := clientV2.Loadbalancers.ListenerDelete(ctx, listenerID)
			if err != nil {
				return diag.FromErr(err)
			}

			taskID := results.Tasks[0]
			err = utilV2.WaitForTaskComplete(ctx, &clientV2, taskID, LBListenerDeleteTimeout)
			if err != nil {
				return diag.FromErr(err)
			}

			opts := edgecloudV2.ListenerCreateRequest{
				Name:             newListener["name"].(string),
				Protocol:         edgecloudV2.LoadbalancerListenerProtocol(newListener["protocol"].(string)),
				ProtocolPort:     newListener["protocol_port"].(int),
				LoadbalancerID:   d.Id(),
				InsertXForwarded: newListener["insert_x_forwarded"].(bool),
				SecretID:         newListener["secret_id"].(string),
			}
			sniSecretIDRaw := newListener["sni_secret_id"].([]interface{})
			if len(sniSecretIDRaw) != 0 {
				sniSecretID := make([]string, len(sniSecretIDRaw))
				for i, s := range sniSecretIDRaw {
					sniSecretID[i] = s.(string)
				}
				opts.SNISecretID = sniSecretID
			}

			_, err = utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Loadbalancers.ListenerCreate, &opts, &clientV2, LBListenerCreateTimeout)
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			opts := &edgecloudV2.ListenerUpdateRequest{
				Name:     newListener["name"].(string),
				SecretID: newListener["secret_id"].(string),
			}
			sniSecretIDRaw := newListener["sni_secret_id"].([]interface{})
			sniSecretID := make([]string, len(sniSecretIDRaw))
			for i, s := range sniSecretIDRaw {
				sniSecretID[i] = s.(string)
			}
			opts.SNISecretID = sniSecretID

			task, _, err := clientV2.Loadbalancers.ListenerUpdate(ctx, listenerID, opts)
			if err != nil {
				return diag.FromErr(err)
			}

			taskID := task.Tasks[0]

			err = utilV2.WaitForTaskComplete(ctx, &clientV2, taskID, LBListenerUpdateTimeout)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		metadataLB := edgecloudV2.Metadata(*meta)
		_, err = clientV2.Loadbalancers.MetadataUpdate(ctx, d.Id(), &metadataLB)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	log.Println("[DEBUG] Finish LoadBalancer updating")

	return resourceLoadBalancerRead(ctx, d, m)
}

func resourceLoadBalancerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer deleting")
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
	results, resp, err := clientV2.Loadbalancers.Delete(ctx, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of Load Balancer deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, &clientV2, taskID, LoadBalancerDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LoadBalancer deleting")

	return diags
}
