package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
)

const (
	LBListenersPoint        = "lblisteners"
	LBListenerCreateTimeout = 2400
)

func resourceLbListener() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLBListenerCreate,
		ReadContext:   resourceLBListenerRead,
		UpdateContext: resourceLBListenerUpdate,
		DeleteContext: resourceLBListenerDelete,
		Description:   "Represent a load balancer listener. Can not be created without a load balancer. A listener is a process that checks for connection requests using the protocol and port that you configure.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, listenerID, lbID, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set("loadbalancer_id", lbID)
				d.SetId(listenerID)

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
				Description: "The name of the load balancer listener.",
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The uuid for the load balancer.",
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Available values are 'TCP', 'UDP', 'HTTP', 'HTTPS' and 'Terminated HTTPS'.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch types.ProtocolType(v) {
					case types.ProtocolTypeTCP, types.ProtocolTypeUDP, types.ProtocolTypeHTTP, types.ProtocolTypeHTTPS, types.ProtocolTypeTerminatedHTTPS:
						return diag.Diagnostics{}
					case types.ProtocolTypePROXY:
					}
					return diag.Errorf("wrong protocol %s, available values are 'TCP', 'UDP', 'HTTP', 'HTTPS' and 'Terminated HTTPS'.", v)
				},
			},
			"protocol_port": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The port on which the protocol is bound.",
			},
			"insert_x_forwarded": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Insert *-forwarded headers",
				ForceNew:    true,
			},
			"pool_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of pools associated with the load balancer.",
			},
			"operating_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current operational status of the load balancer.",
			},
			"provisioning_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current provisioning status of the load balancer.",
			},
			"secret_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The identifier for the associated secret, typically used for SSL configurations.",
			},
			"sni_secret_id": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "List of secret identifiers used for Server Name Indication (SNI).",
			},
			"allowed_cidrs": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "The allowed CIDRs for listener.",
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

func resourceLBListenerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, LBListenersPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := listeners.CreateOpts{
		Name:             d.Get("name").(string),
		Protocol:         types.ProtocolType(d.Get("protocol").(string)),
		ProtocolPort:     d.Get("protocol_port").(int),
		LoadBalancerID:   d.Get("loadbalancer_id").(string),
		InsertXForwarded: d.Get("insert_x_forwarded").(bool),
	}
	secretID := d.Get("secret_id").(string)
	sniSecretIDRaw := d.Get("sni_secret_id").([]interface{})

	switch opts.Protocol { //nolint: exhaustive
	case types.ProtocolTypeTCP, types.ProtocolTypeUDP, types.ProtocolTypeHTTP, types.ProtocolTypeHTTPS:
		if secretID != "" {
			return diag.Errorf("secret_id parameter can only be used with %s listener protocol type", types.ProtocolTypeTerminatedHTTPS)
		}

		if len(sniSecretIDRaw) > 0 {
			return diag.Errorf("sni_secret_id parameter can only be used with %s listener protocol type", types.ProtocolTypeTerminatedHTTPS)
		}

		if opts.InsertXForwarded && (opts.Protocol == types.ProtocolTypeTCP || opts.Protocol == types.ProtocolTypeUDP || opts.Protocol == types.ProtocolTypeHTTPS) {
			return diag.Errorf(
				"X-Forwarded headers can only be used with %s or %s listener protocol type",
				types.ProtocolTypeHTTP, types.ProtocolTypeTerminatedHTTPS,
			)
		}
	case types.ProtocolTypeTerminatedHTTPS:
		if secretID == "" {
			return diag.Errorf("secret_id parameter is required with %s listener protocol type", types.ProtocolTypeTerminatedHTTPS)
		}
		opts.SecretID = secretID
		if len(sniSecretIDRaw) > 0 {
			opts.SNISecretID = make([]string, len(sniSecretIDRaw))
			for i, s := range sniSecretIDRaw {
				opts.SNISecretID[i] = s.(string)
			}
		}
	default:
		return diag.Errorf("wrong protocol")
	}

	allowedCIRDsRaw := d.Get("allowed_cidrs").([]interface{})
	if len(allowedCIRDsRaw) > 0 {
		opts.AllowedCIDRs = make([]string, len(allowedCIRDsRaw))
		for i, s := range allowedCIRDsRaw {
			opts.AllowedCIDRs[i] = s.(string)
		}
	}

	results, err := listeners.Create(client, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	listenerID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, LBListenerCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		listenerID, err := listeners.ExtractListenerIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve LBListener ID from task info: %w", err)
		}
		return listenerID, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(listenerID.(string))
	resourceLBListenerRead(ctx, d, m)

	log.Printf("[DEBUG] Finish LBListener creating (%s)", listenerID)

	return diags
}

func resourceLBListenerRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, LBListenersPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	lb, err := listeners.Get(client, d.Id()).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("name", lb.Name)
	d.Set("protocol", lb.Protocol.String())
	d.Set("protocol_port", lb.ProtocolPort)
	d.Set("pool_count", lb.PoolCount)
	d.Set("operating_status", lb.OperationStatus.String())
	d.Set("provisioning_status", lb.ProvisioningStatus.String())
	d.Set("secret_id", lb.SecretID)
	d.Set("sni_secret_id", lb.SNISecretID)
	d.Set("allowed_cidrs", lb.AllowedCIDRs)

	fields := []string{"project_id", "region_id", "loadbalancer_id", "insert_x_forwarded"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish LBListener reading")

	return diags
}

func resourceLBListenerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener updating")
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, LBListenersPoint, VersionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	var changed bool
	opts := listeners.UpdateOpts{
		Name: d.Get("name").(string),
	}

	if d.HasChange("name") {
		changed = true
	}

	if d.HasChange("secret_id") {
		if types.ProtocolType(d.Get("protocol").(string)) != types.ProtocolTypeTerminatedHTTPS {
			return diag.Errorf("secret_id parameter can only be used with %s listener protocol type", types.ProtocolTypeTerminatedHTTPS)
		}
		opts.SecretID = d.Get("secret_id").(string)
		changed = true
	}

	if d.HasChange("sni_secret_id") {
		if types.ProtocolType(d.Get("protocol").(string)) != types.ProtocolTypeTerminatedHTTPS {
			return diag.Errorf("sni_secret_id parameter can only be used with %s listener protocol type", types.ProtocolTypeTerminatedHTTPS)
		}
		sniSecretIDRaw := d.Get("sni_secret_id").([]interface{})
		sniSecretID := make([]string, len(sniSecretIDRaw))
		for i, s := range sniSecretIDRaw {
			sniSecretID[i] = s.(string)
		}
		opts.SNISecretID = sniSecretID
		changed = true
	}

	if d.HasChange("allowed_cidrs") {
		allowedCIDRsRaw := d.Get("allowed_cidrs").([]interface{})
		allowedCIDRs := make([]string, len(allowedCIDRsRaw))
		for i, s := range allowedCIDRsRaw {
			allowedCIDRs[i] = s.(string)
		}
		opts.AllowedCIDRs = allowedCIDRs
		changed = true
	}

	if changed {
		_, err = listeners.Update(client, d.Id(), opts).Extract()
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	log.Println("[DEBUG] Finish LBListener updating")

	return resourceLBListenerRead(ctx, d, m)
}

func resourceLBListenerDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, LBListenersPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	results, err := listeners.Delete(client, id).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, LBListenerCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		_, err := listeners.Get(client, id).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete LBListener with ID: %s", id)
		}
		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil, nil
		}
		return nil, fmt.Errorf("extracting Listener resource error: %w", err)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LBListener deleting")

	return diags
}
