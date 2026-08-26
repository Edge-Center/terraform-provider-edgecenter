package dbaas

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/tfutil"
)

const DBaaSDatabaseResource = "edgecenter_dbaas_database"

func resourceDBaaSDatabase() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSDatabaseCreate,
		ReadContext:   resourceDBaaSDatabaseRead,
		DeleteContext: resourceDBaaSDatabaseDelete,
		Description:   "Represent DBaaS database resource.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, clusterID, databaseName, err := tfutil.ImportStringParserExtended(d.Id())
				if err != nil {
					return nil, fmt.Errorf("importing DBaaS database: %w", err)
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set(edgecenter.DBaaSClusterIDField, clusterID)
				d.SetId(databaseName)

				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.DBaaSClusterIDField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			edgecenter.NameField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			edgecenter.DBaaSDatabaseEncodingField: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			edgecenter.DBaaSDatabaseLocaleField: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDBaaSDatabaseCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS database creating")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	createOpts := edgecloudV2.DBaaSDatabaseCreateRequest{
		Name: d.Get(edgecenter.NameField).(string),
	}

	if v, ok := d.GetOk(edgecenter.DBaaSDatabaseEncodingField); ok {
		createOpts.Encoding = v.(string)
	}
	if v, ok := d.GetOk(edgecenter.DBaaSDatabaseLocaleField); ok {
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
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

	_ = d.Set(edgecenter.NameField, databaseName)

	return diag.Diagnostics{}
}

func resourceDBaaSDatabaseDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS database deleting")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(edgecenter.DBaaSClusterIDField).(string)
	databaseName := d.Id()

	_, _, err = clientV2.DBaaS.DatabaseDelete(ctx, clusterID, databaseName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Info(ctx, "Finish of DBaaS database deleting")

	return diag.Diagnostics{}
}
