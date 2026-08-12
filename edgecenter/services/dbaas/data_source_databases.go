package dbaas

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const DBaaSDatabasesDataSource = "edgecenter_dbaas_databases"

func dataSourceDBaaSDatabases() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSDatabasesRead,
		Description: "Represent DBaaS databases data source.",
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.DBaaSClusterIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the DBaaS cluster.",
			},
			edgecenter.NameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the database to filter by.",
			},
			"items": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.NameField: {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceDBaaSDatabasesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "[DEBUG] Start DBaaS databases data source reading")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)

	databases, _, err := clientV2.DBaaS.DatabasesList(ctx, clusterID, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	filterName, hasFilter := d.GetOk(edgecenter.NameField)

	var filtered []edgecloudV2.DBaaSDatabase
	for _, db := range databases {
		if hasFilter && db.Name != filterName.(string) {
			continue
		}
		filtered = append(filtered, db)
	}

	if hasFilter && len(filtered) == 0 {
		return diag.Errorf("DBaaS database with name %s not found in cluster %s", filterName.(string), clusterID)
	}

	items := make([]map[string]interface{}, len(filtered))
	for i, db := range filtered {
		items[i] = map[string]interface{}{
			edgecenter.NameField: db.Name,
		}
	}
	_ = d.Set("items", items)

	d.SetId(clusterID)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Finish DBaaS databases data source reading, found %d databases", len(filtered)))

	return diag.Diagnostics{}
}
