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
	MKaaSClusterIDField    = "cluster_id"

	MKaaSPoolFlavorField       = "flavor"
	MKaaSPoolNodeCountField    = "node_count"
	MKaaSPoolVolumeSizeField   = "volume_size"
	MKaaSPoolVolumeTypeField   = "volume_type"
	MKaaSPoolMaxNodeCountField = "max_node_count"
	MKaaSPoolMinNodeCountField = "min_node_count"

	MKaaSPoolLabelsField = "labels"
	MKaaSPoolTaintsField = "taints"

	MKaaSPoolSecurityGroupIDField = "security_group_id"
	MKaaSPoolStateField           = "state"
	MKaaSPoolStatusField          = "status"
)

func resourceMKaaSPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMKaaSPoolCreate,
		ReadContext:   resourceMKaaSPoolRead,
		UpdateContext: resourceMKaaSPoolUpdate,
		DeleteContext: resourceMKaaSPoolDelete,
		Description:   "Represent MKaaS cluster's pool.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(MKaaSPoolCreateTimeout),
			Read:   schema.DefaultTimeout(MKaaSPoolReadTimeout),
			Update: schema.DefaultTimeout(MKaaSPoolUpdateTimeout),
			Delete: schema.DefaultTimeout(MKaaSPoolDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData,
				meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, poolID, clusterIDStr, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				clusterID, err := strconv.Atoi(clusterIDStr)
				if err != nil {
					return nil, fmt.Errorf("invalid cluster_id %q: %w", clusterIDStr, err)
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
			MKaaSPoolFlavorField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			MKaaSPoolNodeCountField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The current number of nodes in the pool.",
			},
			MKaaSPoolMinNodeCountField: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The minimum number of nodes in the pool.",
			},
			MKaaSPoolMaxNodeCountField: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum number of nodes in the pool.",
			},
			MKaaSPoolVolumeSizeField: {
				Type:     schema.TypeInt,
				Required: true,
				Description: "The size of the control volumes in the cluster, specified in gigabytes (GB)." +
					" Allowed range: `20–1024` GiB.",
				ValidateFunc: validation.IntBetween(20, 1024),
			},
			MKaaSPoolVolumeTypeField: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf("The type of volume. Available values are `%s`,"+
					" `%s`.", edgecloudV2.VolumeTypeStandard, edgecloudV2.VolumeTypeSsdHiIops),
				ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VolumeTypeStandard),
					string(edgecloudV2.VolumeTypeSsdHiIops)}, false),
			},
			MKaaSPoolSecurityGroupIDField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the security group associated with the pool.",
			},
			MKaaSPoolLabelsField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of key-value pairs of labels to apply to the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			MKaaSPoolTaintsField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set of taints to apply to the pool nodes.",
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
						},
					},
				},
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

	createOpts := edgecloudV2.MkaaSPoolCreateRequest{
		Name:       d.Get(NameField).(string),
		Flavor:     d.Get(MKaaSPoolFlavorField).(string),
		NodeCount:  d.Get(MKaaSPoolNodeCountField).(int),
		VolumeSize: d.Get(MKaaSPoolVolumeSizeField).(int),
		VolumeType: edgecloudV2.VolumeType(d.Get(MKaaSPoolVolumeTypeField).(string)),
		Labels:     map[string]string{},
		Taints:     []edgecloudV2.MkaaSTaint{},
	}

	if v, ok := d.GetOk(MKaaSPoolMinNodeCountField); ok {
		val := v.(int)
		createOpts.MinNodeCount = &val
	}
	if v, ok := d.GetOk(MKaaSPoolMaxNodeCountField); ok {
		val := v.(int)
		createOpts.MaxNodeCount = &val
	}
	if v, ok := d.GetOk(MKaaSPoolSecurityGroupIDField); ok {
		sg := v.(string)
		createOpts.SecurityGroupID = &sg
	}
	if v, ok := d.GetOk(MKaaSPoolLabelsField); ok {
		for k, iv := range v.(map[string]interface{}) {
			createOpts.Labels[k] = iv.(string)
		}
	}
	// expand taints from TypeSet
	if raw, ok := d.GetOk(MKaaSPoolTaintsField); ok {
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

	log.Println(fmt.Sprintf("MKaaS Pool create request: %+v", createOpts))

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
	log.Println(fmt.Sprintf("MKaaS Pool id (from taskResult): %.0f", poolID))
	d.SetId(strconv.FormatFloat(poolID, 'f', -1, 64))
	resourceMKaaSPoolRead(ctx, d, m)

	log.Println(fmt.Sprintf("Finish MKaaS creating (%s)",
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
	_ = d.Set(NameField, pool.Name)
	_ = d.Set(MKaaSClusterIDField, clusterID)
	_ = d.Set(MKaaSPoolFlavorField, pool.Flavor)
	_ = d.Set(MKaaSPoolNodeCountField, pool.NodeCount)
	_ = d.Set(MKaaSPoolVolumeSizeField, pool.VolumeSize)
	_ = d.Set(MKaaSPoolVolumeTypeField, string(pool.VolumeType))
	_ = d.Set(MKaaSPoolStateField, pool.State)
	_ = d.Set(MKaaSPoolStatusField, pool.Status)

	// Only set optional fields if they were set in the original configuration
	// Check raw config to see if field was originally set
	rawConfig := d.GetRawConfig()

	// Helper function to check if field was in config
	// Safely check if rawConfig is not null before calling AsValueMap()
	fieldInConfig := func(fieldName string) bool {
		if rawConfig.IsNull() {
			return false
		}
		rawConfigMap := rawConfig.AsValueMap()
		if rawConfigMap == nil {
			return false
		}
		val, ok := rawConfigMap[fieldName]
		return ok && !val.IsNull()
	}

	// For min_node_count - only set if it was in config
	if fieldInConfig(MKaaSPoolMinNodeCountField) {
		// Field was in config, set it from API response
		_ = d.Set(MKaaSPoolMinNodeCountField, pool.MinNodeCount)
	}

	// For max_node_count - only set if it was in config
	if fieldInConfig(MKaaSPoolMaxNodeCountField) {
		// Field was in config, set it from API response
		_ = d.Set(MKaaSPoolMaxNodeCountField, pool.MaxNodeCount)
	}

	// For labels - only set if they were in config
	if fieldInConfig(MKaaSPoolLabelsField) {
		if len(pool.Labels) > 0 {
			// Field was in config and API returned values
			labels := map[string]string{}
			for k, v := range pool.Labels {
				labels[k] = v
			}
			_ = d.Set(MKaaSPoolLabelsField, labels)
		} else {
			// Field was in config but API returned empty, set empty map
			_ = d.Set(MKaaSPoolLabelsField, map[string]string{})
		}
	}

	// For taints - only set if they were in config
	if fieldInConfig(MKaaSPoolTaintsField) {
		if len(pool.Taints) > 0 {
			// Field was in config and API returned values
			_ = d.Set(MKaaSPoolTaintsField, flattenTaints(pool.Taints))
		} else {
			// Field was in config but API returned empty, set empty list
			_ = d.Set(MKaaSPoolTaintsField, []interface{}{})
		}
	}

	return diags
}

func resourceMKaaSPoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("Start MKaaS Pool update")

	clusterID := d.Get(MKaaSClusterIDField).(int)
	poolID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid pool id %q: %w", d.Id(), err))
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	updateReq := edgecloudV2.MkaaSPoolUpdateRequest{}
	needsUpdate := false

	if d.HasChange(NameField) {
		name := d.Get(NameField).(string)
		updateReq.Name = &name
		needsUpdate = true
	}

	if d.HasChange(MKaaSPoolNodeCountField) {
		nodeCount := d.Get(MKaaSPoolNodeCountField).(int)
		updateReq.NodeCount = &nodeCount
		needsUpdate = true
	}

	if !needsUpdate {
		log.Println("No MKaaS Pool fields require update")
		return resourceMKaaSPoolRead(ctx, d, m)
	}

	task, _, err := clientV2.MkaaS.PoolUpdate(ctx, clusterID, poolID, updateReq)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := task.Tasks[0]
	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MKaaSPoolUpdateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("Finish MKaaS Pool update")

	return resourceMKaaSPoolRead(ctx, d, m)
}

func resourceMKaaSPoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("Start MKaaS delete")
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
	log.Println(fmt.Sprintf("Task id (%s)", taskID))
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MKaaSPoolDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete MKaaS Pool with ID: %d", poolID)
	}
	d.SetId("")
	log.Println("Finish of MKaaS Pool deleting")

	return diags
}
