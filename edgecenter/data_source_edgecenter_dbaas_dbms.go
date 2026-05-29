package edgecenter

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDBaaSDBMS() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSDBMSRead,
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
			"items": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":      {Type: schema.TypeInt, Computed: true},
						"type":    {Type: schema.TypeString, Computed: true},
						"version": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
		Description: "List available DBMS (database engines) for DBaaS cluster creation.",
	}
}

func dataSourceDBaaSDBMSRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	dbmsList, _, err := clientV2.DBaaS.DbmsList(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	items := make([]map[string]interface{}, len(dbmsList))
	for i, dbms := range dbmsList {
		items[i] = map[string]interface{}{
			"id":      dbms.ID,
			"type":    dbms.Type,
			"version": dbms.Version,
		}
	}
	_ = d.Set("items", items)

	d.SetId(strconv.Itoa(clientV2.Project))

	return diag.Diagnostics{}
}
