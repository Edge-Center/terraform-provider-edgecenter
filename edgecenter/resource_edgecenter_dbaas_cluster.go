package edgecenter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	DBaaSClusterCreateTimeout = 30 * time.Minute
	DBaaSClusterReadTimeout   = 10 * time.Minute
	DBaaSClusterUpdateTimeout = 30 * time.Minute
	DBaaSClusterDeleteTimeout = 20 * time.Minute
)

func resourceDBaaSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSClusterCreate,
		ReadContext:   resourceDBaaSClusterRead,
		UpdateContext: resourceDBaaSClusterUpdate,
		DeleteContext: resourceDBaaSClusterDelete,
		Description:   "Represent DBaaS cluster resource.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(DBaaSClusterCreateTimeout),
			Read:   schema.DefaultTimeout(DBaaSClusterReadTimeout),
			Update: schema.DefaultTimeout(DBaaSClusterUpdateTimeout),
			Delete: schema.DefaultTimeout(DBaaSClusterDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, clusterID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(clusterID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the DBaaS cluster.",
			},
			DescriptionField: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Description: "The description of the DBaaS cluster.",
			},
			DBaaSClusterHighAvailabilityField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Enable high availability for the cluster.",
			},
			FlavorField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The flavor of the DBaaS cluster.",
			},
			"dbms": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of DBMS (e.g., POSTGRESQL).",
						},
						DBaaSDbmsVersionField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The version of DBMS (e.g., 17.5).",
						},
					},
				},
			},
			"volume": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DBaaSVolumeSizeField: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The size of the volume in GB.",
						},
						DBaaSVolumeTypeField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of the volume (e.g., db_standard).",
						},
					},
				},
			},
			"interface": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NetworkIDField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The network ID for the cluster.",
						},
						SubnetIDField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The subnet ID for the cluster.",
						},
					},
				},
			},
			StatusField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			CreatedAtField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			UpdatedAtField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			DBaaSClusterTaskIDField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			DBaaSClusterConnectionField: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DBaaSClusterHostField: {
							Type:     schema.TypeString,
							Computed: true,
						},
						DBaaSClusterPortField: {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceDBaaSClusterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS cluster creating")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := edgecloudV2.DBaaSClusterCreateRequest{
		Name:             d.Get(NameField).(string),
		Description:      d.Get(DescriptionField).(string),
		HighAvailability: d.Get(DBaaSClusterHighAvailabilityField).(bool),
		Flavor:           d.Get(FlavorField).(string),
	}

	if v, ok := d.GetOk("dbms"); ok {
		dbmsList := v.([]interface{})
		if len(dbmsList) > 0 {
			dbms := dbmsList[0].(map[string]interface{})
			createOpts.DBMS = edgecloudV2.DBaaSDbmsType{
				Type:    dbms[TypeField].(string),
				Version: dbms[DBaaSDbmsVersionField].(string),
			}
		}
	}

	if v, ok := d.GetOk("volume"); ok {
		volList := v.([]interface{})
		if len(volList) > 0 {
			vol := volList[0].(map[string]interface{})
			createOpts.Volume = edgecloudV2.DBaaSVolume{
				Size: vol[DBaaSVolumeSizeField].(int),
				Type: edgecloudV2.VolumeType(vol[DBaaSVolumeTypeField].(string)),
			}
		}
	}

	if v, ok := d.GetOk("interface"); ok {
		ifaceList := v.([]interface{})
		if len(ifaceList) > 0 {
			iface := ifaceList[0].(map[string]interface{})
			createOpts.Interface = edgecloudV2.DBaaSClusterInterface{
				NetworkID: iface[NetworkIDField].(string),
				SubnetID:  iface[SubnetIDField].(string),
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("DBaaS cluster create options: %+v", createOpts))

	cluster, err := utilV2.CreateDBaaSClusterAndWait(ctx, clientV2, createOpts, DBaaSClusterCreateTimeout)
	if err != nil {
		return diag.Errorf("error from creating DBaaS cluster: %s", err)
	}

	d.SetId(cluster.ID)
	tflog.Info(ctx, fmt.Sprintf("DBaaS cluster id = %s", cluster.ID))

	return resourceDBaaSClusterRead(ctx, d, m)
}

func resourceDBaaSClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS cluster reading")

	clusterID := d.Id()
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	cluster, resp, err := clientV2.DBaaS.ClusterGet(ctx, clusterID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing DBaaS cluster %s because resource doesn't exist anymore", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set(RegionIDField, clientV2.Region)
	_ = d.Set(ProjectIDField, clientV2.Project)
	_ = d.Set(NameField, cluster.Name)
	_ = d.Set(DescriptionField, cluster.Description)
	_ = d.Set(DBaaSClusterHighAvailabilityField, cluster.HighAvailability)
	_ = d.Set(FlavorField, cluster.Flavor)
	_ = d.Set(StatusField, cluster.Status)
	_ = d.Set(CreatedAtField, cluster.CreatedAt)
	_ = d.Set(UpdatedAtField, cluster.UpdatedAt)

	if cluster.TaskID != "" {
		_ = d.Set(DBaaSClusterTaskIDField, cluster.TaskID)
	}

	if cluster.DBMS != nil {
		dbms := map[string]interface{}{
			TypeField:             cluster.DBMS.Type,
			DBaaSDbmsVersionField: cluster.DBMS.Version,
		}
		_ = d.Set("dbms", []interface{}{dbms})
	}

	if cluster.Volume != nil {
		vol := map[string]interface{}{
			DBaaSVolumeSizeField: cluster.Volume.Size,
			DBaaSVolumeTypeField: string(cluster.Volume.Type),
		}
		_ = d.Set("volume", []interface{}{vol})
	}

	if cluster.Interface != nil {
		iface := map[string]interface{}{
			NetworkIDField: cluster.Interface.NetworkID,
			SubnetIDField:  cluster.Interface.SubnetID,
		}
		_ = d.Set("interface", []interface{}{iface})
	}

	if cluster.Connection != nil {
		conn := map[string]interface{}{
			DBaaSClusterHostField: cluster.Connection.Host,
			DBaaSClusterPortField: cluster.Connection.Port,
		}
		_ = d.Set(DBaaSClusterConnectionField, []interface{}{conn})
	}

	tflog.Debug(ctx, fmt.Sprintf("DBaaS cluster read complete: %+v", cluster))

	return diag.Diagnostics{}
}

func resourceDBaaSClusterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS cluster update")

	clusterID := d.Id()
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := edgecloudV2.DBaaSClusterUpdateRequest{}

	if d.HasChange(NameField) {
		name := d.Get(NameField).(string)
		updateOpts.Name = name
	}
	if d.HasChange(DescriptionField) {
		desc := d.Get(DescriptionField).(string)
		updateOpts.Description = desc
	}
	if d.HasChange(FlavorField) {
		flavor := d.Get(FlavorField).(string)
		updateOpts.Flavor = flavor
	}

	if d.HasChange("volume") {
		if v, ok := d.GetOk("volume"); ok {
			volList := v.([]interface{})
			if len(volList) > 0 {
				vol := volList[0].(map[string]interface{})
				updateOpts.Volume = &edgecloudV2.DBaaSVolume{
					Size: vol[DBaaSVolumeSizeField].(int),
					Type: edgecloudV2.VolumeType(vol[DBaaSVolumeTypeField].(string)),
				}
			}
		}
	}

	tasks, _, err := clientV2.DBaaS.ClusterUpdate(ctx, clusterID, updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(tasks.Tasks) == 0 {
		return diag.Errorf("no tasks returned for cluster update")
	}

	taskID := tasks.Tasks[0]
	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, DBaaSClusterUpdateTimeout)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] task %s not found, assuming update completed: %s", taskID, err))
	}

	tflog.Info(ctx, "Finish DBaaS cluster update")

	return resourceDBaaSClusterRead(ctx, d, m)
}

func resourceDBaaSClusterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS cluster delete")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Id()
	tflog.Info(ctx, fmt.Sprintf("DBaaS cluster id = %s", clusterID))

	results, _, err := clientV2.DBaaS.ClusterDelete(ctx, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(results.Tasks) == 0 {
		return diag.Errorf("no tasks returned for cluster deletion")
	}

	taskID := results.Tasks[0]
	tflog.Info(ctx, fmt.Sprintf("Task id (%s)", taskID))
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, DBaaSClusterDeleteTimeout)
	if err != nil {
		_, resp, getErr := clientV2.DBaaS.ClusterGet(ctx, clusterID)
		if getErr != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] DBaaS cluster %s already deleted, task %s not found", clusterID, taskID))
			d.SetId("")
			return diag.Diagnostics{}
		}
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete DBaaS cluster with ID: %s", clusterID)
	}

	d.SetId("")
	tflog.Info(ctx, "Finish of DBaaS cluster deleting")

	return diag.Diagnostics{}
}
