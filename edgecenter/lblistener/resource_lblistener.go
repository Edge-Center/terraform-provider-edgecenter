package lblistener

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func ResourceEdgeCenterLbListener() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterLbListenerCreate,
		ReadContext:   resourceEdgeCenterLbListenerRead,
		UpdateContext: resourceEdgeCenterLbListenerUpdate,
		DeleteContext: resourceEdgeCenterLbListenerDelete,
		Description: `A listener is a process that checks for connection requests using the protocol and port that you configure.
Can not be created without a load balancer.`,
		Schema: lblistenerSchema(),

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
			protocol := edgecloud.LoadbalancerListenerProtocol(diff.Get("protocol").(string))

			if diff.HasChange("secret_id") {
				if protocol != edgecloud.ListenerProtocolTerminatedHTTPS {
					return fmt.Errorf(
						"secret_id parameter can only be used with %s listener protocol type",
						edgecloud.ListenerProtocolTerminatedHTTPS,
					)
				}
			}

			if diff.HasChange("sni_secret_id") {
				if protocol != edgecloud.ListenerProtocolTerminatedHTTPS {
					return fmt.Errorf(
						"sni_secret_id parameter can only be used with %s listener protocol type",
						edgecloud.ListenerProtocolTerminatedHTTPS,
					)
				}
			}

			return nil
		},
	}
}

func resourceEdgeCenterLbListenerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.ListenerCreateRequest{
		Name:             d.Get("name").(string),
		LoadbalancerID:   d.Get("loadbalancer_id").(string),
		Protocol:         edgecloud.LoadbalancerListenerProtocol(d.Get("protocol").(string)),
		ProtocolPort:     d.Get("protocol_port").(int),
		InsertXForwarded: d.Get("insert_x_forwarded").(bool),
	}

	secretID := d.Get("secret_id").(string)
	sniSecretIDRaw := d.Get("sni_secret_id").([]interface{})

	switch opts.Protocol {
	case edgecloud.ListenerProtocolTCP, edgecloud.ListenerProtocolUDP, edgecloud.ListenerProtocolHTTP, edgecloud.ListenerProtocolHTTPS:
		if secretID != "" {
			return diag.Errorf("secret_id parameter can only be used with %s listener protocol type", edgecloud.ListenerProtocolTerminatedHTTPS)
		}

		if len(sniSecretIDRaw) > 0 {
			return diag.Errorf("sni_secret_id parameter can only be used with %s listener protocol type", edgecloud.ListenerProtocolTerminatedHTTPS)
		}

		if opts.InsertXForwarded && (opts.Protocol == edgecloud.ListenerProtocolTCP || opts.Protocol == edgecloud.ListenerProtocolUDP || opts.Protocol == edgecloud.ListenerProtocolHTTPS) {
			return diag.Errorf(
				"X-Forwarded headers can only be used with %s or %s listener protocol type",
				edgecloud.ListenerProtocolHTTP, edgecloud.ListenerProtocolTerminatedHTTPS,
			)
		}
	case edgecloud.ListenerProtocolTerminatedHTTPS:
		if secretID == "" {
			return diag.Errorf("secret_id parameter is required with %s listener protocol type", edgecloud.ListenerProtocolTerminatedHTTPS)
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

	log.Printf("[DEBUG] Loadbalancer listener create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Loadbalancers.ListenerCreate, opts, client)
	if err != nil {
		return diag.Errorf("error creating loadbalancer listener: %s", err)
	}

	d.SetId(taskResult.Listeners[0])

	log.Printf("[INFO] Listener: %s", d.Id())

	return resourceEdgeCenterLbListenerRead(ctx, d, meta)
}

func resourceEdgeCenterLbListenerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the loadbalancer listener properties for updating the state
	listener, resp, err := client.Loadbalancers.ListenerGet(ctx, d.Id())
	if err != nil {
		// check if the loadbalancer listener no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter Listener (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving loadbalancer listener: %s", err)
	}

	d.Set("name", listener.Name)
	d.Set("protocol", listener.Protocol)
	d.Set("protocol_port", listener.ProtocolPort)
	d.Set("secret_id", listener.SecretID)
	d.Set("sni_secret_id", listener.SNISecretID)
	d.Set("operating_status", listener.OperatingStatus)
	d.Set("provisioning_status", listener.ProvisioningStatus)
	d.Set("pool_count", listener.PoolCount)

	if err := setAllowedCIDRs(ctx, d, listener); err != nil {
		return diag.FromErr(err)
	}

	if err := setInsertHeaders(ctx, d, listener); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceEdgeCenterLbListenerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var changed bool
	opts := &edgecloud.ListenerUpdateRequest{Name: d.Get("name").(string)}

	if d.HasChange("name") {
		changed = true
	}

	if d.HasChange("secret_id") {
		opts.SecretID = d.Get("secret_id").(string)
		changed = true
	}

	if d.HasChange("sni_secret_id") {
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
		task, _, err := client.Loadbalancers.ListenerUpdate(ctx, d.Id(), opts)
		if err != nil {
			return diag.Errorf("Error when changing the loadbalancer listener: %s", err)
		}

		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return diag.Errorf("Error while waiting for loadbalancer listener: %s", err)
		}
	}

	return resourceEdgeCenterLbListenerRead(ctx, d, meta)
}

func resourceEdgeCenterLbListenerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting loadbalancer listener: %s", d.Id())
	task, _, err := client.Loadbalancers.ListenerDelete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("Error deleting loadbalancer listener: %s", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Delete loadbalancer listener task failed with error: %s", err)
	}

	if err = util.ResourceIsDeleted(ctx, client.Loadbalancers.ListenerGet, d.Id()); err != nil {
		return diag.Errorf("Loadbalancer listener with id %s was not deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}
