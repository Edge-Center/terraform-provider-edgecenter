package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	MKaaSPoolReadTimeout   = 10 * time.Minute
	MKaaSPoolCreateTimeout = 60 * time.Minute
	MKaaSPoolUpdateTimeout = 60 * time.Minute
	MKaaSPoolDeleteTimeout = 20 * time.Minute
)

func resourceMKaaSPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMKaaSPoolCreate,
		ReadContext:   resourceMKaaSPoolRead,
		UpdateContext: resourceMKaaSPoolUpdate,
		DeleteContext: resourceMKaaSPoolDelete,
		CustomizeDiff: customMKaaSPoolDiff,
		Description:   "Represent MKaaS cluster's pool.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(MKaaSPoolCreateTimeout),
			Read:   schema.DefaultTimeout(MKaaSPoolReadTimeout),
			Update: schema.DefaultTimeout(MKaaSPoolUpdateTimeout),
			Delete: schema.DefaultTimeout(MKaaSPoolDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData,
				meta interface{},
			) ([]*schema.ResourceData, error) {
				projectID, regionID, poolID, clusterIDStr, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				clusterID, err := strconv.Atoi(clusterIDStr)
				if err != nil {
					return nil, fmt.Errorf("invalid cluster_id %q: %w", clusterIDStr, err)
				}
				_ = d.Set("project_id", projectID)
				_ = d.Set("region_id", regionID)
				_ = d.Set("cluster_id", clusterID)
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
			MKaaSClusterIDField: {
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
			FlavorField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			MKaaSNodeCountField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{MKaaSNodeCountField, MKaaSPoolScalePolicyField},
				Description:  "The number of nodes in the pool.",
			},
			MKaaSPoolCurrentNodeCountField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The current number of nodes in the pool, reflecting the live value from the API (managed by the autoscaler when enabled).",
			},
			MKaaSVolumeSizeField: {
				Type:     schema.TypeInt,
				Required: true,
				Description: "The size of the control volumes in the cluster, specified in gigabytes (GB)." +
					" Allowed range: `20–1024` GiB.",
				ValidateFunc: validation.IntBetween(20, 1024),
			},
			MKaaSVolumeTypeField: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf("The type of volume. Available values are `%s`,"+
					" `%s`.", edgecloudV2.VolumeTypeStandard, edgecloudV2.VolumeTypeSsdHiIops),
				ValidateFunc: validation.StringInSlice([]string{
					string(edgecloudV2.VolumeTypeStandard),
					string(edgecloudV2.VolumeTypeSsdHiIops),
				}, false),
			},
			MKaaSPoolSecurityGroupIDsField: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The list of security group IDs associated with the pool.",
			},
			MKaaSPoolStateField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the pool.",
			},
			MKaaSPoolStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the pool.",
			},
			MKaaSPoolLabelsField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Arbitrary labels assigned to the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			MKaaSPoolTaintsField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Kubernetes taints applied to all nodes in the pool.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"effect": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"NoSchedule",
								"PreferNoSchedule",
								"NoExecute",
							}, false),
						},
					},
				},
			},
			MKaaSPoolScalePolicyField: {
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{MKaaSNodeCountField, MKaaSPoolScalePolicyField},
				Description: "Scale policy for the pool. Presence of `auto_scale` enables the " +
					"Cluster Autoscaler; removing the block disables it.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						MKaaSPoolAutoScaleField: {
							Type:        schema.TypeList,
							Required:    true,
							MaxItems:    1,
							Description: "Auto-scaling configuration for the pool.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									MKaaSPoolMinField: {
										Type:         schema.TypeInt,
										Required:     true,
										Description:  "Minimum number of nodes the autoscaler may scale the pool down to.",
										ValidateFunc: validation.IntAtLeast(1),
									},
									MKaaSPoolMaxField: {
										Type:         schema.TypeInt,
										Required:     true,
										Description:  "Maximum number of nodes the autoscaler may scale the pool up to.",
										ValidateFunc: validation.IntAtLeast(1),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceMKaaSPoolCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS Cluster creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterID := d.Get(MKaaSClusterIDField).(int)

	minVal, maxVal, autoscale := expandScalePolicy(d)

	createOpts := edgecloudV2.MKaaSPoolCreateRequest{
		Name:               d.Get(NameField).(string),
		Flavor:             d.Get(FlavorField).(string),
		VolumeSize:         d.Get(MKaaSVolumeSizeField).(int),
		VolumeType:         edgecloudV2.VolumeType(d.Get(MKaaSVolumeTypeField).(string)),
		Labels:             map[string]string{},
		Taints:             []edgecloudV2.MKaaSTaint{},
		AutoscalingEnabled: autoscale,
	}
	if autoscale {
		createOpts.MinNodeCount = &minVal
		createOpts.MaxNodeCount = &maxVal
	}

	if nc, ok := d.GetOk(MKaaSNodeCountField); ok {
		createOpts.NodeCount = nc.(int)
	} else if autoscale {
		createOpts.NodeCount = minVal
	}
	if v, ok := d.GetOk(MKaaSPoolSecurityGroupIDsField); ok {
		sgs := v.([]interface{})
		ids := make([]string, 0, len(sgs))
		for _, sg := range sgs {
			ids = append(ids, sg.(string))
		}
		createOpts.SecurityGroupIds = ids
	}
	if v, ok := d.GetOk(MKaaSPoolLabelsField); ok {
		for k, iv := range v.(map[string]interface{}) {
			createOpts.Labels[k] = iv.(string)
		}
	}
	// expand taints from TypeSet
	if raw, ok := d.GetOk(MKaaSPoolTaintsField); ok {
		createOpts.Taints = expandTaints(raw.(*schema.Set))
	}

	tflog.Info(ctx, fmt.Sprintf("MKaaS Pool create request: %+v", createOpts))

	results, _, err := clientV2.MkaaS.PoolCreate(ctx, clusterID, createOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MKaaSPoolCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	taskResult, err := utilV2.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		return diag.FromErr(err)
	}

	poolID := taskResult.MkaasPools[0]
	tflog.Info(ctx, fmt.Sprintf("MKaaS Pool id (from taskResult): %.0f", poolID))
	d.SetId(strconv.FormatFloat(poolID, 'f', -1, 64))
	diags = resourceMKaaSPoolRead(ctx, d, m)

	tflog.Info(ctx, fmt.Sprintf("Finish MKaaS creating (%s)",
		strconv.FormatFloat(poolID, 'f', -1, 64)))

	return diags
}

func resourceMKaaSPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("Read MKaaS pool")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(MKaaSClusterIDField).(int)
	poolIDStr := d.Id()
	poolID, err := strconv.Atoi(poolIDStr)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", poolIDStr, err))
	}
	pool, _, err := clientV2.MkaaS.PoolGet(ctx, clusterID, poolID)
	if err != nil {
		return diag.FromErr(err)
	}

	_, _, configAutoscale := expandScalePolicy(d)
	if !pool.AutoscalingEnabled && !configAutoscale {
		_ = d.Set(MKaaSNodeCountField, pool.NodeCount)
	}

	_ = d.Set(NameField, pool.Name)
	_ = d.Set(MKaaSClusterIDField, clusterID)
	_ = d.Set(FlavorField, pool.Flavor)
	_ = d.Set(MKaaSVolumeSizeField, pool.VolumeSize)
	_ = d.Set(MKaaSVolumeTypeField, string(pool.VolumeType))
	_ = d.Set(MKaaSPoolStateField, pool.State)
	_ = d.Set(MKaaSPoolStatusField, pool.Status)
	_ = d.Set(MKaaSPoolSecurityGroupIDsField, pool.SecurityGroupIds)
	_ = d.Set(MKaaSPoolLabelsField, pool.Labels)
	_ = d.Set(MKaaSPoolTaintsField, flattenTaints(pool.Taints))
	_ = d.Set(MKaaSPoolScalePolicyField, flattenScalePolicy(pool))
	_ = d.Set(MKaaSPoolCurrentNodeCountField, pool.NodeCount)

	return diags
}

func resourceMKaaSPoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS Pool update")

	if unsupported := mkaasPoolUnsupportedUpdateChanges(d); len(unsupported) > 0 {
		return diag.Errorf(
			"MKaaS pool update is not supported for these fields: %v. "+
				"Only %q, %q, %q, %q, %q and %q are supported. "+
				"Please revert changes, or recreate the resource if applicable.",
			unsupported,
			NameField,
			MKaaSNodeCountField,
			MKaaSPoolSecurityGroupIDsField,
			MKaaSPoolLabelsField,
			MKaaSPoolTaintsField,
			MKaaSPoolScalePolicyField,
		)
	}

	clusterID := d.Get(MKaaSClusterIDField).(int)

	poolID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", d.Id(), err))
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	waitForTask := func(taskResp *edgecloudV2.TaskResponse) error {
		if taskResp == nil || len(taskResp.Tasks) == 0 {
			return nil
		}
		return utilV2.WaitForTaskComplete(ctx, clientV2, taskResp.Tasks[0], MKaaSPoolUpdateTimeout)
	}

	if d.HasChange(NameField) {
		name := d.Get(NameField).(string)

		tflog.Info(ctx, "Updating MKaaS pool name", map[string]interface{}{
			"pool_id": poolID,
			"name":    name,
		})

		req := edgecloudV2.MKaaSPoolUpdateNameRequest{
			Name: &name,
		}

		taskResp, _, err := clientV2.MkaaS.PoolUpdateName(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	minVal, maxVal, autoscaleNow := expandScalePolicy(d)

	if d.HasChange(MKaaSPoolScalePolicyField) {
		req := edgecloudV2.MKaaSPoolUpdateAutoscalingRequest{
			EnableAutoscaling: &autoscaleNow,
		}
		fields := map[string]interface{}{
			"pool_id": poolID,
			"enabled": autoscaleNow,
		}
		if autoscaleNow {
			req.MinNodeCount = &minVal
			req.MaxNodeCount = &maxVal
			fields["min"] = minVal
			fields["max"] = maxVal
		}

		tflog.Info(ctx, "Updating MKaaS pool autoscaling", fields)

		taskResp, _, err := clientV2.MkaaS.PoolUpdateAutoscaling(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(MKaaSNodeCountField) && !autoscaleNow {
		nodeCount := d.Get(MKaaSNodeCountField).(int)

		tflog.Info(ctx, "Updating MKaaS pool node count", map[string]interface{}{
			"pool_id":    poolID,
			"node_count": nodeCount,
		})

		req := edgecloudV2.MKaaSPoolUpdateScaleRequest{
			NodeCount: &nodeCount,
		}

		taskResp, _, err := clientV2.MkaaS.PoolUpdateNodeCount(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(MKaaSPoolSecurityGroupIDsField) {
		raw := d.Get(MKaaSPoolSecurityGroupIDsField).([]interface{})
		ids := make([]string, 0, len(raw))
		for _, v := range raw {
			ids = append(ids, v.(string))
		}

		tflog.Info(ctx, "Updating MKaaS pool client security groups", map[string]interface{}{
			"pool_id":              poolID,
			"security_group_ids":   ids,
			"security_group_count": len(ids),
		})

		req := edgecloudV2.MKaaSPoolUpdateSecurityGroupsRequest{
			SecurityGroupIds: ids,
		}

		taskResp, _, err := clientV2.MkaaS.PoolUpdateSecurityGroups(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(MKaaSPoolLabelsField) {
		raw := d.Get(MKaaSPoolLabelsField).(map[string]interface{})
		labels := map[string]string{}
		for k, v := range raw {
			labels[k] = v.(string)
		}

		tflog.Info(ctx, "Updating MKaaS pool labels", map[string]interface{}{
			"pool_id": poolID,
			"labels":  labels,
		})

		req := edgecloudV2.MKaaSPoolUpdateLabelsRequest{
			Labels: labels,
		}

		taskResp, _, err := clientV2.MkaaS.PoolUpdateLabels(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(MKaaSPoolTaintsField) {
		taints := expandTaints(d.Get(MKaaSPoolTaintsField).(*schema.Set))

		tflog.Info(ctx, "Updating MKaaS pool taints", map[string]interface{}{
			"pool_id": poolID,
			"taints":  taints,
		})

		req := edgecloudV2.MKaaSPoolUpdateTaintsRequest{
			Taints: taints,
		}

		taskResp, _, err := clientV2.MkaaS.PoolUpdateTaints(ctx, clusterID, poolID, req)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := waitForTask(taskResp); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Info(ctx, "Finish MKaaS Pool update")

	return resourceMKaaSPoolRead(ctx, d, m)
}

func resourceMKaaSPoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS delete")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(MKaaSClusterIDField).(int)
	poolID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", d.Id(), err))
	}

	results, _, err := clientV2.MkaaS.PoolDelete(ctx, clusterID, poolID)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	tflog.Info(ctx, fmt.Sprintf("Task id (%s)", taskID))
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MKaaSPoolDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete MKaaS Pool with ID: %d", poolID)
	}
	d.SetId("")
	tflog.Info(ctx, "Finish of MKaaS Pool deleting")

	return diags
}
