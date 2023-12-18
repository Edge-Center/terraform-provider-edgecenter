package edgecenter

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/pools"
)

func dataSourceK8s() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceK8sRead,
		Description: "Represent k8s cluster with one default pool.",
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
				Description: "The uuid of the Kubernetes cluster.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Kubernetes cluster.",
			},
			"fixed_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fixed network (uuid) associated with the Kubernetes cluster.",
			},
			"fixed_subnet": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Subnet (uuid) associated with the fixed network.",
			},
			"auto_healing_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether auto-healing is enabled for the Kubernetes cluster.",
			},
			"master_lb_floating_ip_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Flag indicating if the master LoadBalancer should have a floating IP.",
			},
			"keypair": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"pool": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Configuration details of the node pool in the Kubernetes cluster.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
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
							Type:     schema.TypeString,
							Computed: true,
						},
						"docker_volume_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"uuid": {
							Type:     schema.TypeString,
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
				},
			},
			"node_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total number of nodes in the Kubernetes cluster.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the Kubernetes cluster.",
			},
			"status_reason": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The reason for the current status of the Kubernetes cluster, if ERROR.",
			},
			"master_addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IP addresses for master nodes in the Kubernetes cluster.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"node_addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of IP addresses for worker nodes in the Kubernetes cluster.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"container_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The container runtime version used in the Kubernetes cluster.",
			},
			"api_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API endpoint address for the Kubernetes cluster.",
			},
			"user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User identifier associated with the Kubernetes cluster.",
			},
			"discovery_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL used for node discovery within the Kubernetes cluster.",
			},
			"health_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Overall health status of the Kubernetes cluster.",
			},
			"health_status_reason": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"faults": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"master_flavor_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Identifier for the master node flavor in the Kubernetes cluster.",
			},
			"cluster_template_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Template identifier from which the Kubernetes cluster was instantiated.",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the Kubernetes cluster.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes cluster was updated.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes cluster was created.",
			},
			"certificate_authority_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The certificate_authority_data field from the Kubernetes cluster config.",
			},
		},
	}
}

func dataSourceK8sRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get("cluster_id").(string)
	cluster, err := clusters.Get(client, clusterID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cluster.UUID)

	d.Set("name", cluster.Name)
	d.Set("fixed_network", cluster.FixedNetwork)
	d.Set("fixed_subnet", cluster.FixedSubnet)
	d.Set("master_lb_floating_ip_enabled", cluster.FloatingIPEnabled)
	d.Set("keypair", cluster.KeyPair)
	d.Set("node_count", cluster.NodeCount)
	d.Set("status", cluster.Status)
	d.Set("status_reason", cluster.StatusReason)

	masterAddresses := make([]string, len(cluster.MasterAddresses))
	for i, addr := range cluster.MasterAddresses {
		masterAddresses[i] = addr.String()
	}
	if err := d.Set("master_addresses", masterAddresses); err != nil {
		return diag.FromErr(err)
	}

	nodeAddresses := make([]string, len(cluster.NodeAddresses))
	for i, addr := range cluster.NodeAddresses {
		nodeAddresses[i] = addr.String()
	}
	if err := d.Set("node_addresses", nodeAddresses); err != nil {
		return diag.FromErr(err)
	}

	d.Set("container_version", cluster.ContainerVersion)
	d.Set("api_address", cluster.APIAddress.String())
	d.Set("user_id", cluster.UserID)
	d.Set("discovery_url", cluster.DiscoveryURL.String())

	d.Set("health_status", cluster.HealthStatus)
	if err := d.Set("health_status_reason", cluster.HealthStatusReason); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("faults", cluster.Faults); err != nil {
		return diag.FromErr(err)
	}

	d.Set("master_flavor_id", cluster.MasterFlavorID)
	d.Set("cluster_template_id", cluster.ClusterTemplateID)
	d.Set("version", cluster.Version)
	d.Set("updated_at", cluster.UpdatedAt.Format(time.RFC850))
	d.Set("created_at", cluster.CreatedAt.Format(time.RFC850))

	var pool pools.ClusterPool
	for _, p := range cluster.Pools {
		if p.IsDefault {
			pool = p
		}
	}

	p := make(map[string]interface{})
	p["uuid"] = pool.UUID
	p["name"] = pool.Name
	p["flavor_id"] = pool.FlavorID
	p["min_node_count"] = pool.MinNodeCount
	p["max_node_count"] = pool.MaxNodeCount
	p["node_count"] = pool.NodeCount
	p["docker_volume_type"] = pool.DockerVolumeType.String()
	p["docker_volume_size"] = pool.DockerVolumeSize
	p["stack_id"] = pool.StackID
	p["created_at"] = pool.CreatedAt.Format(time.RFC850)

	if err := d.Set("pool", []interface{}{p}); err != nil {
		return diag.FromErr(err)
	}

	getConfigResult, err := clusters.GetConfig(client, clusterID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterConfig, err := parseK8sConfig(getConfigResult.Config)
	if err != nil {
		return diag.Errorf("failed to parse k8s config: %s", err)
	}

	certificateAuthorityData := clusterConfig.Clusters[0].Cluster.CertificateAuthorityData
	if err := d.Set("certificate_authority_data", certificateAuthorityData); err != nil {
		return diag.Errorf("couldn't get certificate_authority_data: %s", err)
	}

	log.Println("[DEBUG] Finish K8s reading")

	return diags
}
