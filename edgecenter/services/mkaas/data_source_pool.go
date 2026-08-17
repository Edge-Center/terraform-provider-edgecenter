package mkaas

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const MKaaSPoolDataSource = "edgecenter_mkaas_pool"

func dataSourceMKaaSPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMKaaSPoolRead,
		Description: "Represent MKaaS cluster's pool.",
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The numeric id of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The numeric id of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.MKaaSClusterIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The id of the Kubernetes cluster this pool belongs to.",
			},
			edgecenter.MKaaSPoolIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The id of the Kubernetes pool within the cluster.",
			},
			edgecenter.NameField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Kubernetes pool.",
			},
			edgecenter.FlavorField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			edgecenter.MKaaSNodeCountField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The current number of nodes in the pool.",
			},
			edgecenter.MKaaSPoolCurrentNodeCountField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The current number of nodes in the pool, reflecting the live value from the API (managed by the autoscaler when enabled).",
			},
			edgecenter.MKaaSVolumeSizeField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the volumes used by nodes in the pool, specified in gigabytes (GB).",
			},
			edgecenter.MKaaSVolumeTypeField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of volume used by nodes in the pool.",
			},
			edgecenter.MKaaSPoolLabelsField: {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Arbitrary labels assigned to the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			edgecenter.MKaaSPoolTaintsField: {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Kubernetes taints applied to all nodes in the pool.",
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
						"effect": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			edgecenter.MKaaSPoolSecurityGroupIDsField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of security group IDs attached to the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			edgecenter.MKaaSPoolStateField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the pool.",
			},
			edgecenter.MKaaSPoolStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the pool.",
			},
			edgecenter.MKaaSPoolScalePolicyField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Scale policy of the pool. Populated only when autoscaling is enabled.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.MKaaSPoolAutoScaleField: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Auto-scaling configuration of the pool.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									edgecenter.MKaaSPoolMinNodeCountField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Minimum number of nodes the autoscaler may scale the pool down to.",
									},
									edgecenter.MKaaSPoolMaxNodeCountField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Maximum number of nodes the autoscaler may scale the pool up to.",
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

func dataSourceMKaaSPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.MKaaSClusterIDField).(int)
	poolID := d.Get(edgecenter.MKaaSPoolIDField).(int)

	pool, _, err := clientV2.MkaaS.PoolGet(ctx, clusterID, poolID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get MKaaS pool %d in cluster %d: %w", poolID, clusterID, err))
	}

	d.SetId(strconv.Itoa(pool.ID))
	_ = d.Set(edgecenter.MKaaSClusterIDField, clusterID)
	_ = d.Set(edgecenter.MKaaSPoolIDField, pool.ID)
	_ = d.Set(edgecenter.NameField, pool.Name)
	_ = d.Set(edgecenter.FlavorField, pool.Flavor)
	_ = d.Set(edgecenter.MKaaSNodeCountField, pool.NodeCount)
	_ = d.Set(edgecenter.MKaaSPoolCurrentNodeCountField, pool.NodeCount)
	_ = d.Set(edgecenter.MKaaSVolumeSizeField, pool.VolumeSize)
	_ = d.Set(edgecenter.MKaaSVolumeTypeField, string(pool.VolumeType))
	_ = d.Set(edgecenter.MKaaSPoolSecurityGroupIDsField, pool.SecurityGroupIds)
	_ = d.Set(edgecenter.MKaaSPoolStateField, pool.State)
	_ = d.Set(edgecenter.MKaaSPoolStatusField, pool.Status)
	_ = d.Set(edgecenter.MKaaSPoolLabelsField, pool.Labels)
	_ = d.Set(edgecenter.MKaaSPoolTaintsField, flattenTaints(pool.Taints))
	_ = d.Set(edgecenter.MKaaSPoolScalePolicyField, flattenScalePolicy(pool))

	return nil
}
