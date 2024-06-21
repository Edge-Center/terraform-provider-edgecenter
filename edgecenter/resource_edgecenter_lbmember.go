package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	minWeight             = 0
	maxWeight             = 256
	LBMemberCreateTimeout = 2400 * time.Second
	LBMemberUpdateTimeout = 2400 * time.Second
	LBMemberDeleteTimeout = 2400 * time.Second
)

func resourceLBMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLBMemberCreate,
		ReadContext:   resourceLBMemberRead,
		UpdateContext: resourceLBMemberUpdate,
		DeleteContext: resourceLBMemberDelete,
		Description:   "Represent load balancer member",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, memberID, lbPoolID, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set("pool_id", lbPoolID)
				d.SetId(memberID)

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
			"pool_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The uuid for the load balancer pool.",
			},
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The IP address of the load balancer pool member.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
			},
			"protocol_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The port on which the member listens for requests.",
			},
			"weight": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "A weight value between 0 and 256, determining the distribution of requests among the members of the pool.",
				ValidateDiagFunc: func(val interface{}, path cty.Path) diag.Diagnostics {
					v := val.(int)
					if v >= minWeight && v <= maxWeight {
						return nil
					}
					return diag.Errorf("Valid values: %d to %d got: %d", minWeight, maxWeight, v)
				},
			},
			"subnet_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid of the subnet in which the pool member is located.",
			},
			"instance_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The uuid of the instance (amphora) associated with the pool member.",
			},
			"operating_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current operating status of the pool member.",
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

func resourceLBMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBMember creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	opts := &edgecloudV2.PoolMemberCreateRequest{
		Address:      net.ParseIP(d.Get("address").(string)),
		ProtocolPort: d.Get("protocol_port").(int),
		Weight:       d.Get("weight").(int),
		SubnetID:     d.Get("subnet_id").(string),
		InstanceID:   d.Get("instance_id").(string),
	}

	poolID := d.Get("pool_id").(string)
	results, _, err := clientV2.Loadbalancers.PoolMemberCreate(ctx, poolID, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, &clientV2, taskID, LBMemberCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	poolMember, err := utilV2.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		return diag.FromErr(err)
	}
	pmID := poolMember.Members[0]

	d.SetId(pmID)
	resourceLBMemberRead(ctx, d, m)

	log.Printf("[DEBUG] Finish LBMember creating (%s)", pmID)

	return diags
}

func resourceLBMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBMember reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	poolID := d.Get("pool_id").(string)

	pool, _, err := clientV2.Loadbalancers.PoolGet(ctx, poolID)
	if err != nil {
		return diag.FromErr(err)
	}

	mid := d.Id()
	for _, pm := range pool.Members {
		if mid == pm.ID {
			d.Set("address", pm.Address.String())
			d.Set("protocol_port", pm.ProtocolPort)
			d.Set("weight", pm.Weight)
			d.Set("subnet_id", pm.SubnetID)
			d.Set("instance_id", pm.InstanceID)
			d.Set("operating_status", pm.OperatingStatus)
		}
	}

	fields := []string{"project_id", "region_id"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish LBMember reading)")

	return diags
}

func resourceLBMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBMember updating")
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	poolID := d.Get("pool_id").(string)

	pool, _, err := clientV2.Loadbalancers.PoolGet(ctx, poolID)
	if err != nil {
		return diag.FromErr(err)
	}

	members := make([]edgecloudV2.PoolMemberCreateRequest, len(pool.Members))
	for i, pm := range pool.Members {
		if pm.ID != d.Id() {
			members[i] = edgecloudV2.PoolMemberCreateRequest{
				Address:      pm.Address,
				ProtocolPort: pm.ProtocolPort,
				Weight:       pm.Weight,
				SubnetID:     pm.SubnetID,
				InstanceID:   pm.InstanceID,
				ID:           pm.ID,
			}
			continue
		}

		members[i] = edgecloudV2.PoolMemberCreateRequest{
			Address:      net.ParseIP(d.Get("address").(string)),
			ProtocolPort: d.Get("protocol_port").(int),
			Weight:       d.Get("weight").(int),
			SubnetID:     d.Get("subnet_id").(string),
			InstanceID:   d.Get("instance_id").(string),
			ID:           d.Id(),
		}
	}

	opts := &edgecloudV2.PoolUpdateRequest{Name: pool.Name, Members: members}

	results, _, err := clientV2.Loadbalancers.PoolUpdate(ctx, pool.ID, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, &clientV2, taskID, LBMemberUpdateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish LBMember updating")

	return resourceLBMemberRead(ctx, d, m)
}

func resourceLBMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBMember deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	mid := d.Id()
	pid := d.Get("pool_id").(string)

	results, resp, err := clientV2.Loadbalancers.PoolMemberDelete(ctx, pid, mid)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of LBMember deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, &clientV2, taskID, LBMemberDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LBMember deleting")

	return diags
}
