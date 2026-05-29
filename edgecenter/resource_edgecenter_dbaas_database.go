package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func resourceDBaaSDatabase() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSDatabaseCreate,
		ReadContext:   resourceDBaaSDatabaseRead,
		DeleteContext: resourceDBaaSDatabaseDelete,
		Description:   "Represent DBaaS database resource.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, clusterID, databaseName, err := ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set(DBaaSClusterIDField, clusterID)
				d.SetId(databaseName)

				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			DBaaSClusterIDField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			NameField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			DBaaSDatabaseEncodingField: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			DBaaSDatabaseLocaleField: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDBaaSDatabaseCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS database creating")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(DBaaSClusterIDField).(string)
	createOpts := edgecloudV2.DBaaSDatabaseCreateRequest{
		Name: d.Get(NameField).(string),
	}

	if v, ok := d.GetOk(DBaaSDatabaseEncodingField); ok {
		createOpts.Encoding = v.(string)
	}
	if v, ok := d.GetOk(DBaaSDatabaseLocaleField); ok {
		createOpts.Locale = v.(string)
	}

	_, _, err = clientV2.DBaaS.DatabaseCreate(ctx, clusterID, createOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createOpts.Name)
	tflog.Info(ctx, fmt.Sprintf("DBaaS database id = %s", d.Id()))

	return resourceDBaaSDatabaseRead(ctx, d, m)
}

func resourceDBaaSDatabaseRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS database reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(DBaaSClusterIDField).(string)
	databaseName := d.Id()

	databases, _, err := clientV2.DBaaS.DatabasesList(ctx, clusterID, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	for _, db := range databases {
		if db.Name == databaseName {
			found = true
			break
		}
	}

	if !found {
		tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing DBaaS database %s because resource doesn't exist anymore", d.Id()))
		d.SetId("")
		return nil
	}

	_ = d.Set(NameField, databaseName)

	return diag.Diagnostics{}
}

func resourceDBaaSDatabaseDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS database deleting")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(DBaaSClusterIDField).(string)
	databaseName := d.Id()

	_, _, err = clientV2.DBaaS.DatabaseDelete(ctx, clusterID, databaseName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Info(ctx, "Finish of DBaaS database deleting")

	return diag.Diagnostics{}
}
