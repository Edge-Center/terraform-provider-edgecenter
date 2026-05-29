package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceDBaaSUsers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSUsersRead,
		Description: "Represent DBaaS users data source.",
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
			DBaaSClusterIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the DBaaS cluster.",
			},
			NameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the user to filter by.",
			},
			"items": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NameField: {Type: schema.TypeString, Computed: true},
						DBaaSUserDatabasesField: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceDBaaSUsersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "[DEBUG] Start DBaaS users data source reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(DBaaSClusterIDField).(string)

	users, _, err := clientV2.DBaaS.UsersList(ctx, clusterID, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	filterName, hasFilter := d.GetOk(NameField)

	var filtered []edgecloudV2.DBaaSUser
	for _, u := range users {
		if hasFilter && u.Name != filterName.(string) {
			continue
		}
		filtered = append(filtered, u)
	}

	items := make([]map[string]interface{}, len(filtered))
	for i, u := range filtered {
		databases := make([]string, len(u.Databases))
		for j, db := range u.Databases {
			databases[j] = db.Name
		}
		items[i] = map[string]interface{}{
			NameField:               u.Name,
			DBaaSUserDatabasesField: databases,
		}
	}
	_ = d.Set("items", items)

	d.SetId(clusterID)

	tflog.Debug(ctx, fmt.Sprintf("[DEBUG] Finish DBaaS users data source reading, found %d users", len(filtered)))

	return diag.Diagnostics{}
}
