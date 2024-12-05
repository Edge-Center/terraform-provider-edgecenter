package edgecenter

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	K8sPoint         = "k8s/clusters"
	K8sCreateTimeout = 3600
)

var k8sCreateTimeout = time.Second * time.Duration(K8sCreateTimeout)

func resourceK8s() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This resource is deprecated and will be removed in the next major version. Resource \"edgecenter_k8s\" unavailable.",
		CreateContext:      resourceK8sCreate,
		ReadContext:        resourceK8sRead,
		UpdateContext:      resourceK8sUpdate,
		DeleteContext:      resourceK8sDelete,
		Description:        "Represent k8s cluster with one default pool. \n\n **WARNING:** Resource \"edgecenter_k8s\" is deprecated and unavailable.",
		Timeouts: &schema.ResourceTimeout{
			Create: &k8sCreateTimeout,
			Update: &k8sCreateTimeout,
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, k8sID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(k8sID)

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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kubernetes cluster.",
			},
			"fixed_network": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Fixed network (uuid) associated with the Kubernetes cluster.",
			},
			"fixed_subnet": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Subnet (uuid) associated with the fixed network. Ensure there's a router on this subnet.",
			},
			"auto_healing_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates whether auto-healing is enabled for the Kubernetes cluster. true by default.",
			},
			"master_lb_floating_ip_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag indicating if the master LoadBalancer should have a floating IP.",
			},
			"pods_ip_pool": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "IP pool to be used for pods within the Kubernetes cluster.",
			},
			"services_ip_pool": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "IP pool to be used for services within the Kubernetes cluster.",
			},
			"keypair": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the keypair",
			},
			"pool": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				MinItems:    1,
				Description: "Configuration details of the node pool in the Kubernetes cluster.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"flavor_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"min_node_count": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"max_node_count": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"node_count": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"docker_volume_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Available value is 'standard', 'ssd_hiiops', 'cold', 'ultra'.",
						},
						"docker_volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
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
				Required:    true,
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
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceK8sCreate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s\" is deprecated and unavailable"))
}

func resourceK8sRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s\" is deprecated and unavailable"))
}

func resourceK8sUpdate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s\" is deprecated and unavailable"))
}

func resourceK8sDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_k8s\" is deprecated and unavailable"))
}
