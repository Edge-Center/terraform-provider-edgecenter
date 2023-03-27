package edgecenter

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/pools"
)

func dataSourceK8sPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceK8sPoolRead,
		Description: "Represent k8s cluster's pool.",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"pool_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"is_default": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"flavor_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"min_node_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"max_node_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"node_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"docker_volume_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Available value is 'standard', 'ssd_hiiops', 'cold', 'ultra'.",
			},
			"docker_volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceK8sPoolRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s pool reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get("cluster_id").(string)
	poolID := d.Get("pool_id").(string)

	pool, err := pools.Get(client, clusterID, poolID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(pool.UUID)

	d.Set("name", pool.Name)
	d.Set("cluster_id", clusterID)
	d.Set("is_default", pool.IsDefault)
	d.Set("flavor_id", pool.FlavorID)
	d.Set("min_node_count", pool.MinNodeCount)
	d.Set("max_node_count", pool.MaxNodeCount)
	d.Set("node_count", pool.NodeCount)
	d.Set("docker_volume_type", pool.DockerVolumeType.String())
	d.Set("docker_volume_size", pool.DockerVolumeSize)
	d.Set("stack_id", pool.StackID)
	d.Set("created_at", pool.CreatedAt.Format(time.RFC850))

	log.Println("[DEBUG] Finish K8s pool reading")

	return diags
}
