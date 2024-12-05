package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceK8sPool() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This resource is deprecated and will be removed in the next major version. Resource \"edgecenter_k8s_pool\" unavailable.",
		CreateContext:      resourceK8sPoolCreate,
		ReadContext:        resourceK8sPoolRead,
		UpdateContext:      resourceK8sPoolUpdate,
		DeleteContext:      resourceK8sPoolDelete,
		Description:        "Represent k8s cluster's pool. \n\n **WARNING:** Resource \"edgecenter_k8s_pool\" is deprecated and unavailable.",
		Timeouts: &schema.ResourceTimeout{
			Create: &k8sCreateTimeout,
			Update: &k8sCreateTimeout,
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
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The uuid of the Kubernetes cluster this pool belongs to.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kubernetes pool.",
			},
			"flavor_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the flavor used for nodes in this pool, e.g. g1-standard-2-4.",
			},
			"min_node_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The minimum number of nodes in the pool.",
			},
			"max_node_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The maximum number of nodes the pool can scale to.",
			},
			"node_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The current number of nodes in the pool.",
			},
			"docker_volume_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The type of volume used for the Docker containers. Available values are 'standard', 'ssd_hiiops', 'cold', and 'ultra'.",
			},
			"docker_volume_size": {
				Type:        schema.TypeInt,
				Optional:    true,
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
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceK8sPoolCreate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s_pool\" is deprecated and unavailable"))
}

func resourceK8sPoolRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s_pool\" is deprecated and unavailable"))
}

func resourceK8sPoolUpdate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s_pool\" is deprecated and unavailable"))
}

func resourceK8sPoolDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s_pool\" is deprecated and unavailable"))
}
