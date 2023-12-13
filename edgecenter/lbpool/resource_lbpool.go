package lbpool

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func ResourceEdgeCenterLbPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterLbPoolCreate,
		ReadContext:   resourceEdgeCenterLbPoolRead,
		UpdateContext: resourceEdgeCenterLbPoolUpdate,
		DeleteContext: resourceEdgeCenterLbPoolDelete,
		Description:   `A pool is a list of virtual machines to which the listener will redirect incoming traffic`,
		Schema:        lbpoolSchema(),
	}
}

func resourceEdgeCenterLbPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	sessionPersistence := converter.ListInterfaceToLoadbalancerSessionPersistence(d.Get("session_persistence").([]interface{}))
	healthMonitor := converter.ListInterfaceToHealthMonitor(d.Get("healthmonitor").([]interface{}))

	opts := &edgecloud.PoolCreateRequest{
		LoadbalancerPoolCreateRequest: edgecloud.LoadbalancerPoolCreateRequest{
			Name:                  d.Get("name").(string),
			Protocol:              edgecloud.LoadbalancerPoolProtocol(d.Get("protocol").(string)),
			LoadbalancerAlgorithm: edgecloud.LoadbalancerAlgorithm(d.Get("lb_algorithm").(string)),
			LoadbalancerID:        d.Get("loadbalancer_id").(string),
			ListenerID:            d.Get("listener_id").(string),
			TimeoutClientData:     d.Get("timeout_client_data").(int),
			TimeoutMemberData:     d.Get("timeout_member_data").(int),
			TimeoutMemberConnect:  d.Get("timeout_member_connect").(int),
			SessionPersistence:    sessionPersistence,
			HealthMonitor:         healthMonitor,
		},
	}

	log.Printf("[DEBUG] Loadbalancer pool create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Loadbalancers.PoolCreate, opts, client)
	if err != nil {
		return diag.Errorf("error creating loadbalancer pool: %s", err)
	}

	d.SetId(taskResult.Pools[0])

	log.Printf("[INFO] Pool: %s", d.Id())

	return resourceEdgeCenterLbPoolRead(ctx, d, meta)
}

func resourceEdgeCenterLbPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the loadbalancer pool properties for updating the state
	pool, resp, err := client.Loadbalancers.PoolGet(ctx, d.Id())
	if err != nil {
		// check if the loadbalancer pool no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter Pool (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving loadbalancer pool: %s", err)
	}

	d.Set("name", pool.Name)
	d.Set("lb_algorithm", pool.LoadbalancerAlgorithm)
	d.Set("protocol", pool.Protocol)
	d.Set("timeout_member_connect", pool.TimeoutMemberConnect)
	d.Set("timeout_member_data", pool.TimeoutMemberData)
	d.Set("timeout_client_data", pool.TimeoutClientData)
	d.Set("provisioning_status", pool.ProvisioningStatus)
	d.Set("operating_status", pool.OperatingStatus)

	if len(pool.Loadbalancers) > 0 {
		d.Set("loadbalancer_id", pool.Loadbalancers[0].ID)
	}

	if len(pool.Listeners) > 0 {
		d.Set("listener_id", pool.Listeners[0].ID)
	}

	if err := setHealthMonitor(ctx, d, pool); err != nil {
		return diag.FromErr(err)
	}

	if err := setSessionPersistence(ctx, d, pool); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceEdgeCenterLbPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var changed bool
	opts := &edgecloud.PoolUpdateRequest{
		Name:          d.Get("name").(string),
		HealthMonitor: converter.ListInterfaceToHealthMonitor(d.Get("healthmonitor").([]interface{})),
	}

	if d.HasChange("name") || d.HasChange("healthmonitor") {
		changed = true
	}

	if d.HasChange("timeout_client_data") {
		opts.TimeoutClientData = d.Get("timeout_client_data").(int)
		changed = true
	}

	if d.HasChange("timeout_member_data") {
		opts.TimeoutMemberData = d.Get("timeout_member_data").(int)
		changed = true
	}

	if d.HasChange("timeout_member_connect") {
		opts.TimeoutMemberConnect = d.Get("timeout_member_connect").(int)
		changed = true
	}

	if d.HasChange("lb_algorithm") {
		opts.LoadbalancerAlgorithm = edgecloud.LoadbalancerAlgorithm(d.Get("lb_algorithm").(string))
		changed = true
	}

	if d.HasChange("session_persistence") {
		opts.SessionPersistence = converter.ListInterfaceToLoadbalancerSessionPersistence(d.Get("session_persistence").([]interface{}))
		changed = true
	}

	if changed {
		task, _, err := client.Loadbalancers.PoolUpdate(ctx, d.Id(), opts)
		if err != nil {
			return diag.Errorf("Error when changing the loadbalancer pool: %s", err)
		}

		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return diag.Errorf("Error while waiting for loadbalancer pool update: %s", err)
		}
	}

	return resourceEdgeCenterLbPoolRead(ctx, d, meta)
}

func resourceEdgeCenterLbPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting loadbalancer pool: %s", d.Id())
	task, _, err := client.Loadbalancers.PoolDelete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("Error deleting loadbalancer pool: %s", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Delete loadbalancer pool task failed with error: %s", err)
	}

	if err = util.ResourceIsDeleted(ctx, client.Loadbalancers.PoolGet, d.Id()); err != nil {
		return diag.Errorf("Loadbalancer pool with id %s was not deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}
