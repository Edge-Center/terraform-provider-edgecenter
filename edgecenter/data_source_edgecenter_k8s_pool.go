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

	nodeAddresses := make([]string, len(pool.NodeAddresses))
	for i, na := range pool.NodeAddresses {
		nodeAddresses[i] = na.String()
	}
	d.Set("node_addresses", nodeAddresses)

	poolInstances, err := pools.InstancesAll(client, clusterID, poolID)
	if err != nil {
		return diag.FromErr(err)
	}

	nodeNames := make([]string, len(poolInstances))
	for j, instance := range poolInstances {
		nodeNames[j] = instance.Name
	}
	d.Set("node_names", nodeNames)

	log.Println("[DEBUG] Finish K8s pool reading")

	return diags
}
