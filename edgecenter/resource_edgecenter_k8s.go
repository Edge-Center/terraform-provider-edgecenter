package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/pools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
)

const (
	K8sPoint         = "k8s/clusters"
	K8sCreateTimeout = 3600
)

var k8sCreateTimeout = time.Second * time.Duration(K8sCreateTimeout)

func resourceK8s() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceK8sCreate,
		ReadContext:   resourceK8sRead,
		UpdateContext: resourceK8sUpdate,
		DeleteContext: resourceK8sDelete,
		Description:   "Represent k8s cluster with one default pool.",
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
			"external_dns_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates whether external DNS is enabled for the Kubernetes cluster. false by default.",
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

func resourceK8sCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := clusters.CreateOpts{
		Name:                      d.Get("name").(string),
		Version:                   d.Get("version").(string),
		FixedNetwork:              d.Get("fixed_network").(string),
		FixedSubnet:               d.Get("fixed_subnet").(string),
		KeyPair:                   d.Get("keypair").(string),
		AutoHealingEnabled:        d.Get("auto_healing_enabled").(bool),
		ExternalDNSEnabled:        d.Get("external_dns_enabled").(bool),
		MasterLBFloatingIPEnabled: d.Get("master_lb_floating_ip_enabled").(bool),
	}

	if podsIP, ok := d.GetOk("pods_ip_pool"); ok {
		eccidr, err := parseCIDRFromString(podsIP.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.PodsIPPool = &eccidr
	}

	if svcIP, ok := d.GetOk("services_ip_pool"); ok {
		eccidr, err := parseCIDRFromString(svcIP.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.ServicesIPPool = &eccidr
	}

	poolRaw := d.Get("pool").([]interface{})
	pool := poolRaw[0].(map[string]interface{})

	poolNodeCount := pool["node_count"].(int)
	maxNodeCount := pool["max_node_count"].(int)
	optPool := pools.CreateOpts{
		Name:         pool["name"].(string),
		FlavorID:     pool["flavor_id"].(string),
		NodeCount:    &poolNodeCount,
		MinNodeCount: pool["min_node_count"].(int),
		MaxNodeCount: &maxNodeCount,
	}

	dockerVolumeSize := pool["docker_volume_size"].(int)
	if dockerVolumeSize != 0 {
		optPool.DockerVolumeSize = &dockerVolumeSize
	}

	dockerVolumeType := pool["docker_volume_type"].(string)
	if dockerVolumeType != "" {
		optPool.DockerVolumeType = volumes.VolumeType(dockerVolumeType)
	}

	opts.Pools = []pools.CreateOpts{optPool}
	results, err := clusters.Create(client, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	k8sID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		k8sID, err := clusters.ExtractClusterIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve k8s ID from task info: %w", err)
		}
		return k8sID, nil
	},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(k8sID.(string))
	resourceK8sRead(ctx, d, m)

	log.Printf("[DEBUG] Finish K8s creating (%s)", k8sID)

	return diags
}

func resourceK8sRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	clientK8S, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Id()
	cluster, err := clusters.Get(clientK8S, clusterID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", cluster.Name)
	d.Set("external_dns_enabled", cluster.ExternalDNSEnabled)
	d.Set("fixed_network", cluster.FixedNetwork)
	d.Set("fixed_subnet", cluster.FixedSubnet)
	d.Set("master_lb_floating_ip_enabled", cluster.FloatingIPEnabled)
	d.Set("node_count", cluster.NodeCount)
	d.Set("status", cluster.Status)
	d.Set("status_reason", cluster.StatusReason)

	clientKeypairs, err := CreateClient(provider, d, KeypairsPoint, VersionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	keypairInfo, err := keypairs.Get(clientKeypairs, cluster.KeyPair).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("keypair", keypairInfo.Name)

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

	fields := []string{"region_id", "auto_healing_enabled", "pods_ip_pool", "services_ip_pool"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish K8s reading")

	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s updating")
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("pool") {
		poolRaw := d.Get("pool").([]interface{})[0]
		pool := poolRaw.(map[string]interface{})

		clusterID := d.Id()
		poolID := pool["uuid"].(string)

		if d.HasChanges("pool.0.name", "pool.0.min_node_count", "pool.0.max_node_count") {
			updateOpts := pools.UpdateOpts{
				Name:         pool["name"].(string),
				MinNodeCount: pool["min_node_count"].(int),
				MaxNodeCount: pool["max_node_count"].(int),
			}
			results, err := pools.Update(client, clusterID, poolID, updateOpts).Extract()
			if err != nil {
				return diag.FromErr(err)
			}

			taskID := results.Tasks[0]
			_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
				_, err := pools.Get(client, clusterID, poolID).Extract()
				if err != nil {
					return nil, fmt.Errorf("cannot get pool with ID: %s. Error: %w", poolID, err)
				}
				return nil, nil
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}

		if d.HasChange("pool.0.node_count") {
			resizeOpts := clusters.ResizeOpts{
				NodeCount: pool["node_count"].(*int),
			}
			results, err := clusters.Resize(client, clusterID, poolID, resizeOpts).Extract()
			if err != nil {
				return diag.FromErr(err)
			}

			taskID := results.Tasks[0]
			_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
				_, err := pools.Get(client, clusterID, poolID).Extract()
				if err != nil {
					return nil, fmt.Errorf("cannot get pool with ID: %s. Error: %w", poolID, err)
				}
				return nil, nil
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceK8sRead(ctx, d, m)
}

func resourceK8sDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	results, err := clusters.Delete(client, id).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		_, err := clusters.Get(client, id).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete k8s cluster with ID: %s", id)
		}
		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil, nil
		}
		return nil, fmt.Errorf("extracting Cluster resource error: %w", err)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of K8s deleting")

	return diags
}
