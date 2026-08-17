package mkaas

import (
	"context"
	"fmt"
	"net"
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
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	validationCustom "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/validation"
)

const MKaaSClusterResource = "edgecenter_mkaas_cluster"

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
		CustomizeDiff: customMKaaSClusterDiff,
		Description:   "Represent resourceMKaaSCluster cluster.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(MKaaSClusterCreateTimeout),
			Read:   schema.DefaultTimeout(MKaaSClusterReadTimeout),
			Update: schema.DefaultTimeout(MKaaSClusterUpdateTimeout),
			Delete: schema.DefaultTimeout(MKaaSClusterDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, k8sID, err := edgecenter.ImportStringParser(d.Id())
				if err != nil {
					return nil, fmt.Errorf("importing MKaaS cluster: %w", err)
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(k8sID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either `project_id` or `project_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either `region_id` or `region_name` must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Kubernetes cluster (must be a valid: up to 63 characters, only letters, digits, or '-', and cannot start or end with '-')",
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`),
					"must consist of lower case alphanumeric characters or '-', up to 63 characters, and start and end with an alphanumeric character",
				),
			},
			edgecenter.MKaaSClusterKeypairNameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the SSH keypair.",
			},
			edgecenter.MKaaSClusterPublishKubeAPIToInternet: {
				Type:        schema.TypeBool,
				Description: "Publish kube-api to internet.",
				Optional:    true,
			},
			edgecenter.NetworkIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the network that created the cluster.",
			},
			edgecenter.SubnetIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The id of the subnet that created the cluster.",
			},
			edgecenter.MKaaSClusterControlPlaneField: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.FlavorField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The flavor type of the flavor.",
						},
						edgecenter.MKaaSNodeCountField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The number of control nodes in the cluster (allowed values: `1`, `3`).",
							ValidateFunc: validation.IntInSlice([]int{1, 3}),
						},
						edgecenter.MKaaSVolumeSizeField: {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "The size of the control volumes in the cluster, specified in gigabytes (GB). Allowed range: `30–1024` GiB.",
							ValidateFunc: validation.IntBetween(30, 1024),
						},
						edgecenter.MKaaSVolumeTypeField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  fmt.Sprintf("The type of volumes in the cluster (allowed values: `%s`).", edgecloudV2.VolumeTypeSsdHiIops),
							ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VolumeTypeSsdHiIops)}, false),
						},
						edgecenter.MKaaSClusterVersionField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The version of the Kubernetes cluster (format `vx.xx`). Validated against available versions in the region via API.",
						},
					},
				},
			},
			edgecenter.MKaaSClusterInternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Internal IP address for the Kubernetes cluster.",
			},
			edgecenter.MKaaSClusterExternalIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "External IP address for the Kubernetes cluster.",
			},
			edgecenter.MKaaSClusterCreatedField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the Kubernetes cluster was created.",
			},
			edgecenter.MKaaSClusterProcessingField: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			edgecenter.StatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Kubernetes cluster.",
			},
			edgecenter.MKaaSClusterStageField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Stage of the Kubernetes cluster.",
			},
			edgecenter.MKaaSClusterPodSubnetField: {
				Type:     schema.TypeString,
				Required: true,
				Description: "Pod subnet in CIDR format. Must not overlap with service_subnet and cluster subnet. " +
					"Selected CIDR must be inside RFC1918 ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16",
				ValidateDiagFunc: validationCustom.ValidateCIDRInRanges,
				ForceNew:         true,
			},
			edgecenter.MKaaSClusterServiceSubnetField: {
				Type:     schema.TypeString,
				Required: true,
				Description: "Service subnet in CIDR format. Must not overlap with pod_subnet and cluster subnet. " +
					"Selected CIDR must be inside RFC1918 ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16",
				ValidateDiagFunc: validationCustom.ValidateCIDRInRanges,
				ForceNew:         true,
			},
		},
	}
}

func resourceMKaaSClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS creating")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := edgecloudV2.MKaaSClusterCreateRequest{
		Name:           d.Get(edgecenter.NameField).(string),
		SSHKeyPairName: d.Get(edgecenter.MKaaSClusterKeypairNameField).(string),
		NetworkID:      d.Get(edgecenter.NetworkIDField).(string),
		SubnetID:       d.Get(edgecenter.SubnetIDField).(string),
	}

	if v, ok := d.GetOk(edgecenter.MKaaSClusterPodSubnetField); ok {
		podSubnet := v.(string)
		createOpts.PodSubnet = &podSubnet
	}

	if v, ok := d.GetOk(edgecenter.MKaaSClusterServiceSubnetField); ok {
		serviceSubnet := v.(string)
		createOpts.ServiceSubnet = &serviceSubnet
	}

	if v, ok := d.GetOk(edgecenter.MKaaSClusterPublishKubeAPIToInternet); ok {
		createOpts.PublishKubeAPIToInternet = v.(bool)
	}

	if v, ok := d.GetOk("control_plane"); ok {
		cpList := v.([]interface{})
		if len(cpList) > 0 {
			cp := cpList[0].(map[string]interface{})
			shortVersion := cp[edgecenter.MKaaSClusterVersionField].(string)
			fullVersion, err := resolveK8sVersionFromAPI(ctx, clientV2, clientV2.Region, shortVersion)
			if err != nil {
				return diag.FromErr(err)
			}

			createOpts.ControlPlane = edgecloudV2.ControlPlaneCreateRequest{
				Flavor:     cp[edgecenter.FlavorField].(string),
				NodeCount:  cp[edgecenter.MKaaSNodeCountField].(int),
				VolumeSize: cp[edgecenter.MKaaSVolumeSizeField].(int),
				Version:    fullVersion,
				VolumeType: edgecloudV2.VolumeType(cp[edgecenter.MKaaSVolumeTypeField].(string)),
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	cluster, resp, err := clientV2.MkaaS.ClusterGet(ctx, clusterID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing Mkaas cluster %s because resource doesn't exist anymore", d.Id()))
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	_ = d.Set(edgecenter.RegionIDField, clientV2.Region)
	_ = d.Set(edgecenter.ProjectIDField, clientV2.Project)
	_ = d.Set(edgecenter.NameField, cluster.Name)
	_ = d.Set(edgecenter.MKaaSClusterKeypairNameField, cluster.SSHKeypairName)
	_ = d.Set(edgecenter.NetworkIDField, cluster.NetworkID)
	_ = d.Set(edgecenter.SubnetIDField, cluster.SubnetID)

	cp := map[string]interface{}{
		edgecenter.FlavorField:              cluster.ControlPlane.Flavor,
		edgecenter.MKaaSNodeCountField:      cluster.ControlPlane.NodeCount,
		edgecenter.MKaaSVolumeSizeField:     cluster.ControlPlane.VolumeSize,
		edgecenter.MKaaSVolumeTypeField:     string(cluster.ControlPlane.VolumeType),
		edgecenter.MKaaSClusterVersionField: normalizeVersion(cluster.ControlPlane.Version),
	}
	_ = d.Set(edgecenter.MKaaSClusterControlPlaneField, []interface{}{cp})
	_ = d.Set(edgecenter.MKaaSClusterInternalIPField, cluster.InternalIP)
	_ = d.Set(edgecenter.MKaaSClusterExternalIPField, cluster.ExternalIP)
	_ = d.Set(edgecenter.MKaaSClusterCreatedField, cluster.Created)
	_ = d.Set(edgecenter.MKaaSClusterProcessingField, cluster.Processing)
	_ = d.Set(edgecenter.StatusField, cluster.Status)
	_ = d.Set(edgecenter.MKaaSClusterStageField, cluster.Stage)
	_ = d.Set(edgecenter.MKaaSClusterPodSubnetField, cluster.PodSubnet)
	_ = d.Set(edgecenter.MKaaSClusterServiceSubnetField, cluster.ServiceSubnet)

	return diag.Diagnostics{}
}

func resourceMKaaSClusterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS update")

	if unsupported := mkaasClusterUnsupportedUpdateChanges(d); len(unsupported) > 0 {
		return diag.Errorf(
			"MKaaS cluster update is not supported for these fields: %v. "+
				"Only %q, %q, and %q are supported. "+
				"Please revert changes, or recreate the resource if applicable.",
			unsupported,
			edgecenter.NameField,
			fmt.Sprintf("%s.0.%s", edgecenter.MKaaSClusterControlPlaneField, edgecenter.MKaaSNodeCountField),
			fmt.Sprintf("%s.0.%s", edgecenter.MKaaSClusterControlPlaneField, edgecenter.MKaaSClusterVersionField),
		)
	}

	clusterID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid cluster id: %w", err))
	}

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	controlPlaneNodeCountPath := fmt.Sprintf("%s.%d.%s", edgecenter.MKaaSClusterControlPlaneField, 0, edgecenter.MKaaSNodeCountField)
	controlPlaneVersionPath := fmt.Sprintf("%s.%d.%s", edgecenter.MKaaSClusterControlPlaneField, 0, edgecenter.MKaaSClusterVersionField)
	needsUpdate := d.HasChange(edgecenter.NameField) || d.HasChange(controlPlaneNodeCountPath) || d.HasChange(controlPlaneVersionPath)

	if !needsUpdate {
		tflog.Info(ctx, "No MKaaS cluster fields require update")
		return resourceMKaaSClusterRead(ctx, d, m)
	}

	if d.HasChange(edgecenter.NameField) {
		tflog.Info(ctx, "Updating MKaaS cluster name")

		updateNameReq := edgecloudV2.MKaaSClusterUpdateNameRequest{
			Name: d.Get(edgecenter.NameField).(string),
		}
		task, _, err := clientV2.MkaaS.ClusterUpdateName(ctx, clusterID, updateNameReq)
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := task.Tasks[0]
		err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MKaaSClusterUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}

		tflog.Info(ctx, "Finish MKaaS Cluster name update")
	}

	if d.HasChange(controlPlaneNodeCountPath) {
		tflog.Info(ctx, "Updating MKaaS cluster master node count")

		cpList := d.Get(edgecenter.MKaaSClusterControlPlaneField).([]interface{})
		cp := cpList[0].(map[string]interface{})
		updateMasterNodeCountReq := edgecloudV2.MKaaSClusterUpdateMasterNodeCountRequest{
			MasterNodeCount: cp[edgecenter.MKaaSNodeCountField].(int),
		}

		task, _, err := clientV2.MkaaS.ClusterUpdateMasterNodeCount(ctx, clusterID, updateMasterNodeCountReq)
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := task.Tasks[0]
		err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MKaaSClusterUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}

		tflog.Info(ctx, "Finish MKaaS Cluster master node count update")
	}

	if d.HasChange(controlPlaneVersionPath) {
		tflog.Info(ctx, "Upgrading MKaaS cluster version")

		cpList := d.Get(edgecenter.MKaaSClusterControlPlaneField).([]interface{})
		cp := cpList[0].(map[string]interface{})
		shortVersion := cp[edgecenter.MKaaSClusterVersionField].(string)
		fullVersion, err := resolveK8sVersionFromAPI(ctx, clientV2, clientV2.Region, shortVersion)
		if err != nil {
			return diag.FromErr(err)
		}

		upgradeReq := edgecloudV2.MKaaSClusterUpgradeVersionRequest{
			TargetVersion: fullVersion,
		}

		task, _, err := clientV2.MkaaS.ClusterUpgradeVersion(ctx, clusterID, upgradeReq)
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := task.Tasks[0]
		err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, MKaaSClusterUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}

		tflog.Info(ctx, "Finish MKaaS Cluster version upgrade")
	}

	return resourceMKaaSClusterRead(ctx, d, m)
}

func resourceMKaaSClusterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start MKaaS delete")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
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

func customMKaaSClusterDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	podStr := d.Get(edgecenter.MKaaSClusterPodSubnetField).(string)
	svcStr := d.Get(edgecenter.MKaaSClusterServiceSubnetField).(string)

	if podStr == "" || svcStr == "" {
		return nil
	}

	if podStr == svcStr {
		return fmt.Errorf("pod_subnet and service_subnet cannot be the same CIDR")
	}

	_, podCIDR, err := net.ParseCIDR(podStr)
	if err != nil {
		return fmt.Errorf("invalid pod_subnet: %w", err)
	}

	_, svcCIDR, err := net.ParseCIDR(svcStr)
	if err != nil {
		return fmt.Errorf("invalid service_subnet: %w", err)
	}

	if validationCustom.CidrIntersects(podCIDR, svcCIDR) {
		return fmt.Errorf("pod_subnet (%s) intersects with service_subnet (%s)", podStr, svcStr)
	}

	if !validationCustom.CidrInRFC1918(podCIDR) {
		return fmt.Errorf("pod_subnet %s must belong to private ranges RFC1918", podStr)
	}

	if !validationCustom.CidrInRFC1918(svcCIDR) {
		return fmt.Errorf("service_subnet %s must belong to private ranges RFC1918", svcStr)
	}

	return nil
}
