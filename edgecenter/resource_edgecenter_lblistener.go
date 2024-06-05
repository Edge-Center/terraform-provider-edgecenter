package edgecenter

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	LBListenersPoint        = "lblisteners"
	TimeoutClientData       = "timeout_client_data"
	TimeoutMemberData       = "timeout_member_data"
	TimeoutMemberConnect    = "timeout_member_connect"
	LBListenerCreateTimeout = 2400 * time.Second
	LBListenerUpdateTimeout = 2400 * time.Second
	LBListenerDeleteTimeout = 2400 * time.Second
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
				Description: "Available values are 'TCP', 'UDP', 'HTTP', 'HTTPS' and 'TERMINATED_HTTPS'.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch edgecloudV2.LoadbalancerListenerProtocol(v) {
					case edgecloudV2.ListenerProtocolTCP, edgecloudV2.ListenerProtocolUDP, edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolHTTPS, edgecloudV2.ListenerProtocolTerminatedHTTPS:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong protocol %s, available values are 'TCP', 'UDP', 'HTTP', 'HTTPS' and 'TERMINATED_HTTPS'.", v)
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
			"l7policies": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Set of l7policy uuids attached to this listener.",
				Elem:        &schema.Schema{Type: schema.TypeString},
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
			TimeoutClientData: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The timeout for the frontend client inactivity (in milliseconds).",
			},
			TimeoutMemberData: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The timeout for the backend member inactivity (in milliseconds).",
			},
			TimeoutMemberConnect: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The timeout for the backend member connection (in milliseconds).",
			},
		},
	}
}

func resourceLBListenerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := edgecloudV2.ListenerCreateRequest{
		Name:             d.Get("name").(string),
		Protocol:         edgecloudV2.LoadbalancerListenerProtocol(d.Get("protocol").(string)),
		ProtocolPort:     d.Get("protocol_port").(int),
		LoadbalancerID:   d.Get("loadbalancer_id").(string),
		InsertXForwarded: d.Get("insert_x_forwarded").(bool),
	}
	timeoutCD, ok := d.GetOk(TimeoutClientData)
	if ok {
		typedTimeoutCD := timeoutCD.(int)
		opts.TimeoutClientData = &typedTimeoutCD
	}
	timeoutMD, ok := d.GetOk(TimeoutMemberData)
	if ok {
		typedTimeoutMD := timeoutMD.(int)
		opts.TimeoutMemberData = &typedTimeoutMD
	}
	timeoutMC, ok := d.GetOk(TimeoutMemberConnect)
	if ok {
		typedTimeoutMC := timeoutMC.(int)
		opts.TimeoutMemberConnect = &typedTimeoutMC
	}

	secretID := d.Get("secret_id").(string)
	sniSecretIDRaw := d.Get("sni_secret_id").([]interface{})

	switch opts.Protocol {
	case edgecloudV2.ListenerProtocolTCP, edgecloudV2.ListenerProtocolUDP, edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolHTTPS:
		if secretID != "" {
			return diag.Errorf("secret_id parameter can only be used with %s listener protocol type", edgecloudV2.ListenerProtocolTerminatedHTTPS)
		}

		if len(sniSecretIDRaw) > 0 {
			return diag.Errorf("sni_secret_id parameter can only be used with %s listener protocol type", edgecloudV2.ListenerProtocolTerminatedHTTPS)
		}

		if opts.InsertXForwarded && (opts.Protocol == edgecloudV2.ListenerProtocolTCP || opts.Protocol == edgecloudV2.ListenerProtocolUDP || opts.Protocol == edgecloudV2.ListenerProtocolHTTPS) {
			return diag.Errorf(
				"X-Forwarded headers can only be used with %s or %s listener protocol type",
				edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolTerminatedHTTPS,
			)
		}
	case edgecloudV2.ListenerProtocolTerminatedHTTPS:
		if secretID == "" {
			return diag.Errorf("secret_id parameter is required with %s listener protocol type", edgecloudV2.ListenerProtocolTerminatedHTTPS)
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

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Loadbalancers.ListenerCreate, &opts, clientV2, LBListenerCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	listenerID := taskResult.Listeners[0]

	d.SetId(listenerID)
	resourceLBListenerRead(ctx, d, m)

	log.Printf("[DEBUG] Finish LBListener creating (%s)", listenerID)

	return diags
}

func resourceLBListenerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	listener, _, err := clientV2.Loadbalancers.ListenerGet(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", listener.Name)
	d.Set("protocol", listener.Protocol)
	d.Set("protocol_port", listener.ProtocolPort)
	d.Set("pool_count", listener.PoolCount)
	d.Set("operating_status", listener.OperatingStatus)
	d.Set("provisioning_status", listener.ProvisioningStatus)
	d.Set("secret_id", listener.SecretID)
	d.Set("sni_secret_id", listener.SNISecretID)
	d.Set("allowed_cidrs", listener.AllowedCIDRs)
	d.Set(TimeoutClientData, listener.TimeoutClientData)
	d.Set(TimeoutMemberData, listener.TimeoutMemberData)
	d.Set(TimeoutMemberConnect, listener.TimeoutMemberConnect)

	l7Policies, err := GetListenerL7PolicyUUIDS(ctx, clientV2, listener.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("l7policies", l7Policies)

	fields := []string{"project_id", "region_id", "loadbalancer_id", "insert_x_forwarded"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish LBListener reading")

	return diags
}

func resourceLBListenerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener updating")

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	var changed bool
	opts := edgecloudV2.ListenerUpdateRequest{
		Name: d.Get("name").(string),
	}

	if d.HasChange("name") {
		changed = true
	}

	if d.HasChange("secret_id") {
		if edgecloudV2.LoadbalancerListenerProtocol(d.Get("protocol").(string)) != edgecloudV2.ListenerProtocolTerminatedHTTPS {
			return diag.Errorf("secret_id parameter can only be used with %s listener protocol type", edgecloudV2.ListenerProtocolTerminatedHTTPS)
		}
		opts.SecretID = d.Get("secret_id").(string)
		changed = true
	}

	if d.HasChange("sni_secret_id") {
		if edgecloudV2.LoadbalancerListenerProtocol(d.Get("protocol").(string)) != edgecloudV2.ListenerProtocolTerminatedHTTPS {
			return diag.Errorf("sni_secret_id parameter can only be used with %s listener protocol type", edgecloudV2.ListenerProtocolTerminatedHTTPS)
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
		var allowedCIDRs []string
		for _, s := range allowedCIDRsRaw {
			allowedCIDRs = append(allowedCIDRs, s.(string))
		}
		opts.AllowedCIDRs = &allowedCIDRs
		changed = true
	}
	if d.HasChange(TimeoutClientData) {
		timeoutCD, ok := d.GetOk(TimeoutClientData)
		if ok {
			typedTimeoutCD := timeoutCD.(int)
			opts.TimeoutClientData = &typedTimeoutCD
			changed = true
		}
	}
	if d.HasChange(TimeoutMemberData) {
		timeoutMD, ok := d.GetOk(TimeoutMemberData)
		if ok {
			typedTimeoutMD := timeoutMD.(int)
			opts.TimeoutMemberData = &typedTimeoutMD
			changed = true
		}
	}
	if d.HasChange(TimeoutMemberConnect) {
		timeoutMC, ok := d.GetOk(TimeoutMemberConnect)
		if ok {
			typedTimeoutMC := timeoutMC.(int)
			opts.TimeoutMemberConnect = &typedTimeoutMC
			changed = true
		}
	}

	if changed {
		task, _, err := clientV2.Loadbalancers.ListenerUpdate(ctx, d.Id(), &opts)
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := task.Tasks[0]

		err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, LBListenerUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	log.Println("[DEBUG] Finish LBListener updating")

	return resourceLBListenerRead(ctx, d, m)
}

func resourceLBListenerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	results, _, err := clientV2.Loadbalancers.ListenerDelete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, LBListenerDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete LBListener with ID: %s", id)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LBListener deleting")

	return diags
}
