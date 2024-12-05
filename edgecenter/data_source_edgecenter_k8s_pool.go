package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceK8sPool() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source \"edgecenter_k8s_pool\" unavailable.",
		ReadContext:        dataSourceK8sPoolRead,
		Description:        "Represent k8s cluster's pool.\n\n **WARNING:** Data source \"edgecenter_k8s_pool\" is deprecated and unavailable.",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"pool_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The uuid of the Kubernetes pool within the cluster.",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The uuid of the Kubernetes cluster this pool belongs to.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Kubernetes pool.",
			},
			"is_default": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether this pool is the default pool in the cluster.",
			},
			"flavor_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the flavor used for nodes in this pool.",
			},
			"min_node_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The minimum number of nodes in the pool.",
			},
			"max_node_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum number of nodes the pool can scale to.",
			},
			"node_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The current number of nodes in the pool.",
			},
			"docker_volume_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of volume used for the Docker containers. Available values are 'standard', 'ssd_hiiops', 'cold', and 'ultra'.",
			},
			"docker_volume_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the volume used for Docker containers, in gigabytes.",
			},
			"stack_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the underlying infrastructure stack used by this pool.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes pool was created.",
			},
			"node_addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of IP addresses of nodes within the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"node_names": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of names of nodes within the pool.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceK8sPoolRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("data source \"edgecenter_k8s_pool\" is deprecated and unavailable"))
}
