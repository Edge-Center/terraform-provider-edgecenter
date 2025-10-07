package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	K8sPoint              = "k8s/clusters"
	K8sCreateTimeout      = 3600
	MkaasKeypairNameField = "ssh_keypair_name"

	MkaasControlPlaneField = "control_plane"
	MkaasFlavorField       = "flavor"
	MkaasNodeCountField    = "node_count"
	MkaasVolumeSizeField   = "volume_size"
	MkaasVolumeTypeField   = "volume_type"
	MkaasVersionField      = "version"

	MkaasInternalIPField = "internal_ip"
	MkaasExternalIPField = "external_ip"
	MkaasCreatedField    = "created"
	MkaasProcessingField = "processing"
	MkaasStatusField     = "status"
)

var k8sCreateTimeout = time.Second * time.Duration(K8sCreateTimeout)

func resourceMkaasCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMkaasCreate,
		ReadContext:   resourceMkaasRead,
		UpdateContext: resourceMkaasUpdate,
		DeleteContext: resourceMkaasDelete,
		Description:   "Represent k8s cluster.",
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
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either `project_id` or `project_name` must be specified.",
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
				Description:  "The uuid of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kubernetes cluster (must be a valid: up to 63 characters, only letters, digits, or '-', and cannot start or end with '-')",
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`),
					"must consist of lower case alphanumeric characters or '-', up to 63 characters, and start and end with an alphanumeric character",
				),
			},
			MkaasKeypairNameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the SSH keypair.",
			},
			NetworkIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the network that created the cluster.",
			},
			SubnetIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the subnet that created the cluster.",
			},
			MkaasControlPlaneField: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						MkaasFlavorField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The flavor type of the flavor.",
						},
						MkaasNodeCountField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The number of control nodes in the cluster (allowed values: `1`, `3`, `5`).",
							ValidateFunc: validation.IntInSlice([]int{1, 3, 5}),
						},
						MkaasVolumeSizeField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The size of the control volumes in the cluster, specified in gigabytes (GB). Allowed range: `20–1024` GiB.",
							ValidateFunc: validation.IntBetween(20, 1024),
						},
						MkaasVolumeTypeField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  fmt.Sprintf("The type of volumes in the cluster (allowed values: `%s`).", edgecloudV2.VolumeTypeSsdHiIops),
							ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VolumeTypeSsdHiIops)}, false),
						},
						MkaasVersionField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The version of the Kubernetes cluster (format `vx.xx.x`).",
							ValidateFunc: validation.StringInSlice([]string{"v1.31.0"}, false),
						},
					},
				},
			},
			MkaasInternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Internal IP address for the Kubernetes cluster.",
			},
			MkaasExternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "External IP address for the Kubernetes cluster.",
			},
			MkaasCreatedField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes cluster was created.",
			},
			MkaasProcessingField: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			MkaasStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Kubernetes cluster.",
			},
		},
	}
}

func resourceMkaasCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8S creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := edgecloudV2.MkaaSClusterCreateRequest{
		Name:           d.Get(NameField).(string),
		SSHKeyPairName: d.Get(MkaasKeypairNameField).(string),
		NetworkID:      d.Get(NetworkIDField).(string),
		SubnetID:       d.Get(SubnetIDField).(string),
	}

	if v, ok := d.GetOk("control_plane"); ok {
		cpList := v.([]interface{})
		if len(cpList) > 0 {
			cp := cpList[0].(map[string]interface{})
			createOpts.ControlPlane = edgecloudV2.ControlPlaneCreateRequest{
				Flavor:     cp[MkaasFlavorField].(string),
				NodeCount:  cp[MkaasNodeCountField].(int),
				VolumeSize: cp[MkaasVolumeSizeField].(int),
				Version:    cp[MkaasVersionField].(string),
				VolumeType: edgecloudV2.VolumeType(cp[MkaasVolumeTypeField].(string)),
			}
		}
	}

	log.Printf("[DEBUG] MKaaS create options: %+v", createOpts)

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.MkaaS.ClusterCreate, createOpts, clientV2, k8sCreateTimeout)
	if err != nil {
		return diag.Errorf("error from creating mkaas: %s", err)
	}

	clusterID := taskResult.MkaasClusters[0]
	log.Printf("[DEBUG] MKaaS id (from taskResult): %s", clusterID)
	d.SetId(clusterID)
	resourceMkaasRead(ctx, d, m)

	log.Printf("[DEBUG] Finish MKaaS creating (%s)", clusterID)

	return diags
}

func resourceMkaasRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS reading")
	var diags diag.Diagnostics

	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		d.SetId("")
		return diag.Errorf("invalid cluster id: %s", err)
	}
	log.Printf("[DEBUG] MKaaS id = %d", clusterID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	cluster, resp, err := clientV2.MkaaS.ClusterGet(ctx, clusterID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Removing Mkaas cluster %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set(RegionIDField, clientV2.Region)
	_ = d.Set(ProjectIDField, clientV2.Project)
	_ = d.Set(NameField, cluster.Name)
	_ = d.Set(MkaasKeypairNameField, cluster.SSHKeypairName)
	_ = d.Set(NetworkIDField, cluster.NetworkID)
	_ = d.Set(SubnetIDField, cluster.SubnetID)

	cp := map[string]interface{}{
		MkaasFlavorField:     cluster.ControlPlane.Flavor,
		MkaasNodeCountField:  cluster.ControlPlane.NodeCount,
		MkaasVolumeSizeField: cluster.ControlPlane.VolumeSize,
		MkaasVolumeTypeField: string(cluster.ControlPlane.VolumeType),
		MkaasVersionField:    cluster.ControlPlane.Version,
	}
	_ = d.Set(MkaasControlPlaneField, []interface{}{cp})
	_ = d.Set(MkaasInternalIPField, cluster.InternalIP)
	_ = d.Set(MkaasExternalIPField, cluster.ExternalIP)
	_ = d.Set(MkaasCreatedField, cluster.Created)
	_ = d.Set(MkaasProcessingField, cluster.Processing)
	_ = d.Set(StatusField, cluster.Status)

	return diags
}

func resourceMkaasUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS update")

	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid cluster id: %w", err))
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(MkaasControlPlaneField) || d.HasChange(NameField) {
		if v, ok := d.GetOk(MkaasControlPlaneField); ok {
			cpList := v.([]interface{})
			if len(cpList) > 0 {
				cp := cpList[0].(map[string]interface{})
				nodeCount := cp[MkaasNodeCountField].(int)

				opts := edgecloudV2.MkaaSClusterUpdateRequest{
					Name:            d.Get(NameField).(string),
					MasterNodeCount: nodeCount,
				}
				if _, _, err := clientV2.MkaaS.ClusterUpdate(ctx, clusterID, opts); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	log.Println("[DEBUG] Finish MKaaS update")

	return resourceMkaasRead(ctx, d, m)
}

func resourceMkaasDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start MKaaS delete")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := strconv.Atoi(d.Id())
	log.Printf("[DEBUG] MKaaS cluster id = %d", clusterID)
	if err != nil {
		d.SetId("")
		return nil
	}

	results, _, err := clientV2.MkaaS.ClusterDelete(ctx, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, k8sCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete MKaaS cluster with ID: %d", clusterID)
	}
	d.SetId("")
	log.Printf("[DEBUG] Finish of MKaaS cluster deleting")

	return diags
}
