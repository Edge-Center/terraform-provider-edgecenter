package edgecenter

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	MKaaSClusterReadTimeout   = 10 * time.Minute
	MKaaSClusterCreateTimeout = 30 * time.Minute
	MKaaSClusterUpdateTimeout = 30 * time.Minute
	MKaaSClusterDeleteTimeout = 20 * time.Minute
)

func resourceMKaaSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMKaaSClusterCreate,
		ReadContext:   resourceMKaaSClusterRead,
		UpdateContext: resourceMKaaSClusterUpdate,
		DeleteContext: resourceMKaaSClusterDelete,
		Description:   "Represent resourceMKaaSCluster cluster.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(MKaaSClusterCreateTimeout),
			Read:   schema.DefaultTimeout(MKaaSClusterReadTimeout),
			Update: schema.DefaultTimeout(MKaaSClusterUpdateTimeout),
			Delete: schema.DefaultTimeout(MKaaSClusterDeleteTimeout),
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
			MKaaSClusterKeypairNameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the SSH keypair.",
			},
			MKaaSClusterPublishKubeAPIToInternet: {
				Type:        schema.TypeBool,
				Description: "Publish kube-api to internet.",
				Optional:    true,
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
			MKaaSClusterControlPlaneField: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FlavorField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The flavor type of the flavor.",
						},
						MKaaSNodeCountField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The number of control nodes in the cluster (allowed values: `1`, `3`).",
							ValidateFunc: validation.IntInSlice([]int{1, 3}),
						},
						MKaaSVolumeSizeField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The size of the control volumes in the cluster, specified in gigabytes (GB). Allowed range: `20–1024` GiB.",
							ValidateFunc: validation.IntBetween(20, 1024),
						},
						MKaaSVolumeTypeField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  fmt.Sprintf("The type of volumes in the cluster (allowed values: `%s`).", edgecloudV2.VolumeTypeSsdHiIops),
							ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VolumeTypeSsdHiIops)}, false),
						},
						MKaaSClusterVersionField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The version of the Kubernetes cluster (format `vx.xx.x`).",
							ValidateFunc: validation.StringInSlice([]string{"v1.31.0"}, false),
						},
					},
				},
			},
			MKaaSClusterInternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Internal IP address for the Kubernetes cluster.",
			},
			MKaaSClusterExternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "External IP address for the Kubernetes cluster.",
			},
			MKaaSClusterCreatedField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes cluster was created.",
			},
			MKaaSClusterProcessingField: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			StatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Kubernetes cluster.",
			},
			MKaaSClusterStateField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "State of the Kubernetes cluster.",
			},
		},
	}
}

func resourceMKaaSClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, fmt.Sprintf("Start MKaaS creating"))

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := edgecloudV2.MKaaSClusterCreateRequest{
		Name:           d.Get(NameField).(string),
		SSHKeyPairName: d.Get(MKaaSClusterKeypairNameField).(string),
		NetworkID:      d.Get(NetworkIDField).(string),
		SubnetID:       d.Get(SubnetIDField).(string),
	}

	if v, ok := d.GetOk(MKaaSClusterPublishKubeAPIToInternet); ok {
		createOpts.PublishKubeAPIToInternet = v.(bool)
	}

	if v, ok := d.GetOk("control_plane"); ok {
		cpList := v.([]interface{})
		if len(cpList) > 0 {
			cp := cpList[0].(map[string]interface{})
			createOpts.ControlPlane = edgecloudV2.ControlPlaneCreateRequest{
				Flavor:     cp[FlavorField].(string),
				NodeCount:  cp[MKaaSNodeCountField].(int),
				VolumeSize: cp[MKaaSVolumeSizeField].(int),
				Version:    cp[MKaaSClusterVersionField].(string),
				VolumeType: edgecloudV2.VolumeType(cp[MKaaSVolumeTypeField].(string)),
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("MKaaS create options: %+v", createOpts))

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.MkaaS.ClusterCreate,
		createOpts, clientV2, MKaaSClusterCreateTimeout)
	if err != nil {
		return diag.Errorf("error from creating mkaas: %s", err)
	}

	clusterID := taskResult.MkaasClusters[0]
	tflog.Info(ctx, fmt.Sprintf("MKaaS id (from taskResult): %.0f", clusterID))
	d.SetId(strconv.FormatFloat(clusterID, 'f', -1, 64))

	diags := resourceMKaaSClusterRead(ctx, d, m)

	tflog.Info(ctx, fmt.Sprintf("Finish MKaaS creating (%.0f)", clusterID))

	return diags
}

func resourceMKaaSClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS reading")

	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("invalid cluster id: %s", err)
	}
	tflog.Info(ctx, fmt.Sprintf("MKaaS id = %d", clusterID))

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	cluster, resp, err := clientV2.MkaaS.ClusterGet(ctx, clusterID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing Mkaas cluster %s because resource doesn't exist anymore", d.Id()))
			return nil
		}

		return diag.FromErr(err)
	}

	_ = d.Set(RegionIDField, clientV2.Region)
	_ = d.Set(ProjectIDField, clientV2.Project)
	_ = d.Set(NameField, cluster.Name)
	_ = d.Set(MKaaSClusterKeypairNameField, cluster.SSHKeypairName)
	_ = d.Set(NetworkIDField, cluster.NetworkID)
	_ = d.Set(SubnetIDField, cluster.SubnetID)

	cp := map[string]interface{}{
		FlavorField:              cluster.ControlPlane.Flavor,
		MKaaSNodeCountField:      cluster.ControlPlane.NodeCount,
		MKaaSVolumeSizeField:     cluster.ControlPlane.VolumeSize,
		MKaaSVolumeTypeField:     string(cluster.ControlPlane.VolumeType),
		MKaaSClusterVersionField: cluster.ControlPlane.Version,
	}
	_ = d.Set(MKaaSClusterControlPlaneField, []interface{}{cp})
	_ = d.Set(MKaaSClusterInternalIPField, cluster.InternalIP)
	_ = d.Set(MKaaSClusterExternalIPField, cluster.ExternalIP)
	_ = d.Set(MKaaSClusterCreatedField, cluster.Created)
	_ = d.Set(MKaaSClusterProcessingField, cluster.Processing)
	_ = d.Set(StatusField, cluster.Status)
	_ = d.Set(MKaaSClusterStateField, cluster.State)

	return diag.Diagnostics{}
}

func resourceMKaaSClusterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS update")

	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid cluster id: %w", err))
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	updateReq := edgecloudV2.MKaaSClusterUpdateRequest{}
	needsUpdate := false

	if d.HasChange(NameField) {
		updateReq.Name = d.Get(NameField).(string)
		needsUpdate = true
	}

	controlPlaneNodeCountPath := fmt.Sprintf("%s.%d.%s", MKaaSClusterControlPlaneField, 0, MKaaSNodeCountField)
	if d.HasChange(controlPlaneNodeCountPath) {
		if v, ok := d.GetOk(MKaaSClusterControlPlaneField); ok {
			cpList := v.([]interface{})
			if len(cpList) > 0 {
				cp := cpList[0].(map[string]interface{})
				updateReq.MasterNodeCount = cp[MKaaSNodeCountField].(int)
				needsUpdate = true
			}
		}
	}

	if !needsUpdate {
		tflog.Info(ctx, "No MKaaS cluster fields require update")
		return resourceMKaaSClusterRead(ctx, d, m)
	}

	task, _, err := clientV2.MkaaS.ClusterUpdate(ctx, clusterID, updateReq)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := task.Tasks[0]
	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MKaaSClusterUpdateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Finish MKaaS Cluster update")

	return resourceMKaaSClusterRead(ctx, d, m)
}

func resourceMKaaSClusterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS delete")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, err := strconv.Atoi(d.Id())
	tflog.Info(ctx, fmt.Sprintf("MKaaS cluster id = %d", clusterID))
	if err != nil {
		d.SetId("")
		return nil
	}

	results, _, err := clientV2.MkaaS.ClusterDelete(ctx, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	tflog.Info(ctx, fmt.Sprintf("Task id (%s)", taskID))
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, MKaaSClusterDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete MKaaS cluster with ID: %d", clusterID)
	}
	d.SetId("")
	tflog.Info(ctx, "Finish of MKaaS cluster deleting")

	return diag.Diagnostics{}
}
