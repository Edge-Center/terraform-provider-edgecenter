package edgecenter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceDBaaSClusters() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSClustersRead,
		Description: "Represent DBaaS cluster data source.",
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
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the DBaaS cluster. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", NameField},
			},
			IDField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The ID of the DBaaS cluster. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", NameField},
			},
			DescriptionField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			StatusField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			DBaaSClusterHighAvailabilityField: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			FlavorField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dbms": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField:             {Type: schema.TypeString, Computed: true},
						DBaaSDbmsVersionField: {Type: schema.TypeString, Computed: true},
					},
				},
			},
			"volume": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DBaaSVolumeSizeField: {Type: schema.TypeInt, Computed: true},
						DBaaSVolumeTypeField: {Type: schema.TypeString, Computed: true},
					},
				},
			},
			"interface": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NetworkIDField: {Type: schema.TypeString, Computed: true},
						SubnetIDField:  {Type: schema.TypeString, Computed: true},
					},
				},
			},
			DBaaSClusterConnectionField: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DBaaSClusterHostField: {Type: schema.TypeString, Computed: true},
						DBaaSClusterPortField: {Type: schema.TypeInt, Computed: true},
					},
				},
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
		},
	}
}

func dataSourceDBaaSClustersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "[DEBUG] Start DBaaS clusters data source reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var (
		cluster   *edgecloudV2.DBaaSCluster
		clusterID string
	)

	if v, ok := d.GetOk(IDField); ok && v.(string) != "" {
		clusterID = v.(string)
	} else {
		name := d.Get(NameField).(string)
		if name == "" {
			return diag.Errorf("either 'id' or 'name' must be specified")
		}

		clusters, _, err := clientV2.DBaaS.ClustersList(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
		if len(clusters) == 0 {
			return diag.Errorf("DBaaS cluster with name %s not found", name)
		}

		var found bool
		for _, c := range clusters {
			if c.Name == name {
				clusterID = c.ID
				found = true
				break
			}
		}

		if !found {
			return diag.Errorf("DBaaS cluster with name %s not found", name)
		}

		if len(clusters) > 1 {
			count := 0
			for _, c := range clusters {
				if c.Name == name {
					count++
				}
			}
			if count > 1 {
				return diag.Errorf("multiple DBaaS clusters found with name %s; please specify 'id'", name)
			}
		}
	}

	var resp *edgecloudV2.Response
	cluster, resp, err = clientV2.DBaaS.ClusterGet(ctx, clusterID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return diag.Errorf("DBaaS cluster %s not found", clusterID)
		}
		return diag.FromErr(err)
	}
	if cluster == nil {
		return diag.Errorf("DBaaS cluster %s: empty response", clusterID)
	}

	d.SetId(cluster.ID)

	_ = d.Set(RegionIDField, cluster.RegionID)
	_ = d.Set(ProjectIDField, cluster.ProjectID)
	_ = d.Set(NameField, cluster.Name)
	_ = d.Set(DescriptionField, cluster.Description)
	_ = d.Set(DBaaSClusterHighAvailabilityField, cluster.HighAvailability)
	_ = d.Set(FlavorField, cluster.Flavor)
	_ = d.Set(StatusField, cluster.Status)

	dbms := map[string]interface{}{
		TypeField:             cluster.DBMS.Type,
		DBaaSDbmsVersionField: cluster.DBMS.Version,
	}
	_ = d.Set("dbms", []interface{}{dbms})

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

	_ = d.Set(CreatedAtField, cluster.CreatedAt)
	_ = d.Set(UpdatedAtField, cluster.UpdatedAt)

	if cluster.TaskID != "" {
		_ = d.Set(DBaaSClusterTaskIDField, cluster.TaskID)
	}

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Finish DBaaS clusters data source reading, cluster ID: %s", cluster.ID))

	return diag.Diagnostics{}
}
