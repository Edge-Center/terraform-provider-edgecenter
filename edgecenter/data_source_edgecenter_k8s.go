package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceK8s() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source \"edgecenter_k8s\" unavailable.",
		ReadContext:        dataSourceK8sRead,
		Description:        "Represent k8s cluster with one default pool.\n\n **WARNING:** Data source \"edgecenter_k8s\" is deprecated and unavailable.",
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

func dataSourceK8sRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s\" is deprecated and unavailable"))
}
