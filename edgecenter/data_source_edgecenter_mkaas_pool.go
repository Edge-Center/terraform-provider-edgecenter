package edgecenter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceMKaaSPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMKaaSPoolRead,
		Description: "Represent MKaaS cluster's pool.",
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The numeric id of the project. Either `project_id` or `project_name` must be specified.",
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
				Description:  "The numeric id of the region. Either `region_id` or `region_name` must be specified.",
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
				Description: "The id of the Kubernetes cluster this pool belongs to.",
			},
			MKaaSPoolIDField: {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The id of the Kubernetes pool within the cluster.",
			},
			NameField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Kubernetes pool.",
			},
			MKaaSPoolFlavorField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			MKaaSPoolNodeCountField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The current number of nodes in the pool.",
			},
			MKaaSPoolVolumeSizeField: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the volumes used by nodes in the pool, specified in gigabytes (GB).",
			},
			MKaaSPoolVolumeTypeField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of volume used by nodes in the pool.",
			},
			MKaaSPoolSecurityGroupIDsField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of security group IDs attached to the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
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

func dataSourceMKaaSPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(MKaaSClusterIDField).(int)
	poolID := d.Get(MKaaSPoolIDField).(int)

	pool, _, err := clientV2.MkaaS.PoolGet(ctx, clusterID, poolID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get MKaaS pool %d in cluster %d: %w", poolID, clusterID, err))
	}

	d.SetId(strconv.Itoa(pool.ID))
	_ = d.Set(MKaaSClusterIDField, clusterID)
	_ = d.Set(MKaaSPoolIDField, pool.ID)
	_ = d.Set(NameField, pool.Name)
	_ = d.Set(MKaaSPoolFlavorField, pool.Flavor)
	_ = d.Set(MKaaSPoolNodeCountField, pool.NodeCount)
	_ = d.Set(MKaaSPoolVolumeSizeField, pool.VolumeSize)
	_ = d.Set(MKaaSPoolVolumeTypeField, string(pool.VolumeType))
	_ = d.Set(MKaaSPoolSecurityGroupIDsField, pool.SecurityGroupIds)
	_ = d.Set(MKaaSPoolStateField, pool.State)
	_ = d.Set(MKaaSPoolStatusField, pool.Status)

	return nil
}
