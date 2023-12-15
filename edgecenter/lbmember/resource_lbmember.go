package lbmember

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func ResourceEdgeCenterLbMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterLbMemberCreate,
		ReadContext:   resourceEdgeCenterLbMemberRead,
		UpdateContext: resourceEdgeCenterLbMemberUpdate,
		DeleteContext: resourceEdgeCenterLbMemberDelete,
		Description: `A Member node represents a physical server that acts as a provider of a service available to a load balancer. 
Does not support concurrent update of multiple members. Update one at a time`,
		Schema: lbmemberSchema(),
	}
}

func resourceEdgeCenterLbMemberCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.PoolMemberCreateRequest{
		Address:      net.ParseIP(d.Get("address").(string)),
		ProtocolPort: d.Get("protocol_port").(int),
		Weight:       d.Get("weight").(int),
		SubnetID:     d.Get("subnet_id").(string),
		InstanceID:   d.Get("instance_id").(string),
		AdminStateUP: d.Get("admin_state_up").(bool),
	}

	log.Printf("[DEBUG] Loadbalancer pool member create configuration: %#v", opts)

	task, _, err := client.Loadbalancers.PoolMemberCreate(ctx, d.Get("pool_id").(string), opts)
	if err != nil {
		return diag.Errorf("error creating loadbalancer pool member: %s", err)
	}

	taskInfo, err := util.WaitAndGetTaskInfo(ctx, client, task.Tasks[0])
	if err != nil {
		return diag.Errorf("error waiting for pool member create: %s", err)
	}

	taskResult, err := util.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		return diag.Errorf("error while extract task result: %s", err)
	}

	d.SetId(taskResult.Members[0])

	log.Printf("[INFO] Pool member: %s", d.Id())

	return resourceEdgeCenterLbMemberRead(ctx, d, meta)
}

func resourceEdgeCenterLbMemberRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	member, err := util.PoolMemberGetByID(ctx, client, d.Get("pool_id").(string), d.Id())
	if errors.Is(err, util.ErrLoadbalancerPoolsMemberNotFound) {
		log.Printf("[WARN] EdgeCenter Pool member (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("address", member.Address.String())
	d.Set("protocol_port", member.ProtocolPort)
	d.Set("weight", member.Weight)
	d.Set("subnet_id", member.SubnetID)
	d.Set("instance_id", member.InstanceID)
	d.Set("operating_status", member.OperatingStatus)

	return nil
}

func resourceEdgeCenterLbMemberUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	pool, _, err := client.Loadbalancers.PoolGet(ctx, d.Get("pool_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	members := make([]edgecloud.PoolMemberCreateRequest, len(pool.Members))
	for i, pm := range pool.Members {
		if pm.ID != d.Id() {
			members[i] = edgecloud.PoolMemberCreateRequest{
				ID:           pm.ID,
				Address:      pm.Address,
				ProtocolPort: pm.ProtocolPort,
				Weight:       pm.Weight,
				SubnetID:     pm.SubnetID,
				InstanceID:   pm.InstanceID,
				AdminStateUP: pm.AdminStateUP,
			}

			continue
		}

		members[i] = edgecloud.PoolMemberCreateRequest{
			ID:           d.Id(),
			Address:      net.ParseIP(d.Get("address").(string)),
			ProtocolPort: d.Get("protocol_port").(int),
			Weight:       d.Get("weight").(int),
			SubnetID:     d.Get("subnet_id").(string),
			InstanceID:   d.Get("instance_id").(string),
			AdminStateUP: d.Get("admin_state_up").(bool),
		}
	}

	opts := &edgecloud.PoolUpdateRequest{
		ID:                    pool.ID,
		Name:                  pool.Name,
		LoadbalancerAlgorithm: pool.LoadbalancerAlgorithm,
		Members:               members,
		TimeoutMemberData:     pool.TimeoutMemberData,
		TimeoutClientData:     pool.TimeoutClientData,
		TimeoutMemberConnect:  pool.TimeoutMemberConnect,
		HealthMonitor: edgecloud.HealthMonitorCreateRequest{
			Type:           pool.HealthMonitor.Type,
			MaxRetries:     pool.HealthMonitor.MaxRetries,
			Delay:          pool.HealthMonitor.Delay,
			Timeout:        pool.HealthMonitor.Timeout,
			ID:             pool.HealthMonitor.ID,
			ExpectedCodes:  pool.HealthMonitor.ExpectedCodes,
			MaxRetriesDown: pool.HealthMonitor.MaxRetriesDown,
			HTTPMethod:     pool.HealthMonitor.HTTPMethod,
			URLPath:        pool.HealthMonitor.URLPath,
		},
	}
	task, _, err := client.Loadbalancers.PoolUpdate(ctx, pool.ID, opts)
	if err != nil {
		return diag.Errorf("Error when changing the loadbalancer pool: %s", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Error while waiting for loadbalancer pool update: %s", err)
	}

	poolAfterUpdate, _, err := client.Loadbalancers.PoolGet(ctx, d.Get("pool_id").(string))
	if err != nil {
		return diag.Errorf("Error when get the loadbalancer pool info after update: %s", err)
	}

	for _, pm := range poolAfterUpdate.Members {
		if pm.Address == nil {
			continue
		}

		if net.IP.Equal(pm.Address, net.ParseIP(d.Get("address").(string))) &&
			pm.ProtocolPort == d.Get("protocol_port").(int) && pm.SubnetID == d.Get("subnet_id").(string) {
			d.SetId(pm.ID)
			break
		}
	}

	return resourceEdgeCenterLbMemberRead(ctx, d, meta)
}

func resourceEdgeCenterLbMemberDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting loadbalancer pool member: %s", d.Id())
	task, _, err := client.Loadbalancers.PoolMemberDelete(ctx, d.Get("pool_id").(string), d.Id())
	if err != nil {
		return diag.Errorf("Error deleting pool member: %s", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Delete pool member task failed with error: %s", err)
	}

	_, err = util.PoolMemberGetByID(ctx, client, d.Get("pool_id").(string), d.Id())
	if !errors.Is(err, util.ErrLoadbalancerPoolsMemberNotFound) {
		return diag.Errorf("Pool member with id %s was not deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}
