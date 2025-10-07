package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	MkaasPoolReadTimeout   = 10 * time.Minute
	MkaasPoolCreateTimeout = 60 * time.Minute
	MkaasPoolUpdateTimeout = 60 * time.Minute
	MkaasPoolDeleteTimeout = 20 * time.Minute
	MkaasClusterIDField    = "cluster_id"

	MkaasPoolFlavorField       = "flavor"
	MkaasPoolNodeCountField    = "node_count"
	MkaasPoolVolumeSizeField   = "volume_size"
	MkaasPoolVolumeTypeField   = "volume_type"
	MkaasPoolMaxNodeCountField = "max_node_count"
	MkaasPoolMinNodeCountField = "min_node_count"

	MkaasPoolLabelsField = "labels"
	MkaasPoolTaintsField = "taints"

	MkaasPoolSecurityGroupIDField = "security_group_id"
	MkaasPoolStateField           = "state"
	MkaasPoolStatusField          = "status"
)

func resourceMkaasPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMkaasPoolCreate,
		ReadContext:   resourceMkaasPoolRead,
		UpdateContext: resourceMkaasPoolUpdate,
		DeleteContext: resourceMkaasPoolDelete,
		Description:   "Represent MKaaS cluster's pool.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(MkaasPoolCreateTimeout),
			Read:   schema.DefaultTimeout(MkaasPoolReadTimeout),
			Update: schema.DefaultTimeout(MkaasPoolUpdateTimeout),
			Delete: schema.DefaultTimeout(MkaasPoolDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, poolID, clusterID, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set("cluster_id", clusterID)
				d.SetId(poolID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			MkaasClusterIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the Kubernetes cluster this pool belongs to.",
			},
			NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kubernetes pool.",
			},
			MkaasPoolFlavorField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			MkaasPoolNodeCountField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The current number of nodes in the pool.",
			},
			MkaasPoolVolumeSizeField: {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "The size of the control volumes in the cluster, specified in gigabytes (GB). Allowed range: `20–1024` GiB.",
				ValidateFunc: validation.IntBetween(20, 1024),
			},
			MkaasPoolVolumeTypeField: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  fmt.Sprintf("The type of volume. Available values are `%s`, `%s`.", edgecloudV2.VolumeTypeStandard, edgecloudV2.VolumeTypeSsdHiIops),
				ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VolumeTypeStandard), string(edgecloudV2.VolumeTypeSsdHiIops)}, false),
			},
			MkaasPoolSecurityGroupIDField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the security group associated with the pool.",
			},
			MkaasPoolStateField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the pool.",
			},
			MkaasPoolStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the pool.",
			},
		},
	}
}

func resourceMkaasPoolCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS Cluster creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(MkaasClusterIDField).(int)

	createOpts := edgecloudV2.MkaaSPoolCreateRequest{
		Name:       d.Get(NameField).(string),
		Flavor:     d.Get(MkaasPoolFlavorField).(string),
		NodeCount:  d.Get(MkaasPoolNodeCountField).(int),
		VolumeSize: d.Get(MkaasPoolVolumeSizeField).(int),
		VolumeType: edgecloudV2.VolumeType(d.Get(MkaasPoolVolumeTypeField).(string)),
		Labels:     map[string]string{},
		Taints:     []edgecloudV2.MkaaSTaint{},
	}

	if v, ok := d.GetOk(MkaasPoolMinNodeCountField); ok {
		val := v.(int)
		createOpts.MinNodeCount = &val
	}
	if v, ok := d.GetOk(MkaasPoolMaxNodeCountField); ok {
		val := v.(int)
		createOpts.MaxNodeCount = &val
	}
	if v, ok := d.GetOk(MkaasPoolSecurityGroupIDField); ok {
		sg := v.(string)
		createOpts.SecurityGroupID = &sg
	}
	if v, ok := d.GetOk(MkaasPoolLabelsField); ok {
		for k, iv := range v.(map[string]interface{}) {
			createOpts.Labels[k] = iv.(string)
		}
	}
	// expand taints from TypeSet
	if raw, ok := d.GetOk(MkaasPoolTaintsField); ok {
		set := raw.(*schema.Set)
		for _, item := range set.List() {
			m := item.(map[string]interface{})
			createOpts.Taints = append(createOpts.Taints, edgecloudV2.MkaaSTaint{
				Key:    m["key"].(string),
				Value:  m["value"].(string),
				Effect: m["effect"].(string),
			})
		}
	}

	log.Printf("[DEBUG] MKaaS Pool create request: %+v", createOpts)

	results, _, err := clientV2.MkaaS.PoolCreate(ctx, clusterID, createOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MkaasPoolCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	taskResult, err := utilV2.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		return diag.FromErr(err)
	}

	poolID := taskResult.MkaasPools[0]
	log.Printf("[DEBUG] MKaaS Pool id (from taskResult): %.0f", poolID)
	d.SetId(strconv.FormatFloat(poolID, 'f', -1, 64))
	resourceMkaasPoolRead(ctx, d, m)

	log.Printf("[DEBUG] Finish MKaaS creating (%s)", strconv.FormatFloat(poolID, 'f', -1, 64))

	return diags
}

func resourceMkaasPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Read MKaaS pool")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(MkaasClusterIDField).(int)
	poolIDStr := d.Id()
	poolID, err := strconv.Atoi(poolIDStr)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", poolIDStr, err))
	}
	pool, _, err := clientV2.MkaaS.PoolGet(ctx, clusterID, poolID)
	_ = d.Set(NameField, pool.Name)
	_ = d.Set(MkaasClusterIDField, clusterID)
	_ = d.Set(MkaasPoolFlavorField, pool.Flavor)
	_ = d.Set(MkaasPoolNodeCountField, pool.NodeCount)
	_ = d.Set(MkaasPoolMinNodeCountField, pool.MinNodeCount)
	_ = d.Set(MkaasPoolMaxNodeCountField, pool.MaxNodeCount)
	_ = d.Set(MkaasPoolVolumeSizeField, pool.VolumeSize)
	_ = d.Set(MkaasPoolVolumeTypeField, string(pool.VolumeType))
	_ = d.Set(MkaasPoolStateField, pool.State)
	_ = d.Set(MkaasPoolStatusField, pool.Status)

	if pool.Labels != nil {
		labels := map[string]string{}
		for k, v := range pool.Labels {
			labels[k] = v
		}
		_ = d.Set(MkaasPoolLabelsField, labels)
	}

	if pool.Taints != nil {
		_ = d.Set(MkaasPoolTaintsField, flattenTaints(pool.Taints))
	} else {
		_ = d.Set(MkaasPoolTaintsField, nil)
	}

	return diags
}

func resourceMkaasPoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS Pool update")

	clusterID := d.Get(MkaasClusterIDField).(int)
	poolID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", d.Id(), err))
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(NameField) || d.HasChange(MkaasPoolNodeCountField) {
		name := d.Get(NameField).(string)
		nodeCount := d.Get(MkaasPoolNodeCountField).(int)
		opts := edgecloudV2.MkaaSPoolUpdateRequest{
			Name:      &name,
			NodeCount: &nodeCount,
		}
		task, _, err := clientV2.MkaaS.PoolUpdate(ctx, clusterID, poolID, opts)
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := task.Tasks[0]

		err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MkaasPoolUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	log.Println("[DEBUG] Finish MKaaS Pool update")

	return resourceMkaasPoolRead(ctx, d, m)
}

func resourceMkaasPoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS delete")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(MkaasClusterIDField).(int)
	poolID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", d.Id(), err))
	}

	results, _, err := clientV2.MkaaS.PoolDelete(ctx, clusterID, poolID)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MkaasPoolDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete MKaaS Pool with ID: %d", clusterID)
	}
	d.SetId("")
	log.Printf("[DEBUG] Finish of MKaaS Pool deleting")

	return diags
}

func flattenTaints(taints []edgecloudV2.MkaaSTaint) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(taints))
	for _, t := range taints {
		result = append(result, map[string]interface{}{
			"key":    t.Key,
			"value":  t.Value,
			"effect": t.Effect,
		})
	}
	return result
}
