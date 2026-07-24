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
	DBaaSBackupCreateTimeout = 30 * time.Minute
	DBaaSBackupReadTimeout   = 10 * time.Minute
	DBaaSBackupUpdateTimeout = 10 * time.Minute
	DBaaSBackupDeleteTimeout = 20 * time.Minute
)

func resourceDBaaSBackup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSBackupCreate,
		ReadContext:   resourceDBaaSBackupRead,
		UpdateContext: resourceDBaaSBackupUpdate,
		DeleteContext: resourceDBaaSBackupDelete,
		Description:   "Represent a manual DBaaS backup resource.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(DBaaSBackupCreateTimeout),
			Read:   schema.DefaultTimeout(DBaaSBackupReadTimeout),
			Update: schema.DefaultTimeout(DBaaSBackupUpdateTimeout),
			Delete: schema.DefaultTimeout(DBaaSBackupDeleteTimeout),
		},
		Importer: &schema.ResourceImporter{StateContext: resourceDBaaSBackupImport},
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type: schema.TypeInt, Optional: true, ForceNew: true,
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type: schema.TypeString, Optional: true, ForceNew: true,
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type: schema.TypeInt, Optional: true, ForceNew: true,
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type: schema.TypeString, Optional: true, ForceNew: true,
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField:                     {Type: schema.TypeString, Required: true},
			DescriptionField:              {Type: schema.TypeString, Optional: true},
			DBaaSClusterIDField:           {Type: schema.TypeString, Required: true, ForceNew: true},
			DBaaSBackupParentIDField:      {Type: schema.TypeString, Optional: true, ForceNew: true},
			DBaaSBackupTypeField:          computedStringSchema(),
			StatusField:                   computedStringSchema(),
			DBaaSBackupSizeField:          {Type: schema.TypeFloat, Computed: true},
			DBaaSBackupIsServiceField:     {Type: schema.TypeBool, Computed: true},
			DBaaSBackupHasChildField:      {Type: schema.TypeBool, Computed: true},
			CreatedAtField:                computedStringSchema(),
			UpdatedAtField:                computedStringSchema(),
			DBaaSBackupFinishedAtField:    computedStringSchema(),
			DBaaSClusterTaskIDField:       computedStringSchema(),
			DBaaSBackupCreatorTaskIDField: computedStringSchema(),
			"dbms": {
				Type: schema.TypeList, Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					TypeField:             computedStringSchema(),
					DBaaSDbmsVersionField: computedStringSchema(),
				}},
			},
		},
	}
}

func resourceDBaaSBackupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS backup creating")
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := edgecloudV2.DBaaSBackupCreateRequest{
		Name:      d.Get(NameField).(string),
		ClusterID: d.Get(DBaaSClusterIDField).(string),
	}

	if v, ok := d.GetOk(DescriptionField); ok {
		createOpts.Description = v.(string)
	}
	if v, ok := d.GetOk(DBaaSBackupParentIDField); ok {
		createOpts.ParentID = v.(string)
	}

	tflog.Debug(ctx, fmt.Sprintf("DBaaS backup create options: %+v", createOpts))

	backup, err := utilV2.CreateDBaaSBackupAndWait(ctx, clientV2, createOpts)
	if err != nil {
		return diag.Errorf("error from creating DBaaS backup: %s", err)
	}

	d.SetId(backup.ID)
	tflog.Info(ctx, fmt.Sprintf("DBaaS backup id = %s", backup.ID))

	return resourceDBaaSBackupRead(ctx, d, m)
}

func resourceDBaaSBackupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS backup reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	backup, resp, err := clientV2.DBaaS.BackupGet(ctx, d.Id(), false)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("[WARN] Removing DBaaS backup %s because resource doesn't exist anymore", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if err := setDBaaSBackupData(d, clientV2, backup); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func resourceDBaaSBackupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS backup update")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.HasChange(NameField) && !d.HasChange(DescriptionField) {
		return resourceDBaaSBackupRead(ctx, d, m)
	}

	updateOpts := edgecloudV2.DBaaSBackupUpdateRequest{}

	if d.HasChange(NameField) {
		name := d.Get(NameField).(string)
		updateOpts.Name = &name
	}
	if d.HasChange(DescriptionField) {
		desc := d.Get(DescriptionField).(string)
		updateOpts.Description = &desc
	}

	_, _, err = clientV2.DBaaS.BackupUpdate(ctx, d.Id(), updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDBaaSBackupRead(ctx, d, m)
}

func resourceDBaaSBackupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Start DBaaS backup delete")
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancel()

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	backupID := d.Id()
	tflog.Info(ctx, fmt.Sprintf("DBaaS backup id = %s", backupID))

	if err := utilV2.DeleteResourceIfExist(ctx, clientV2, clientV2.DBaaS, backupID, d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	tflog.Info(ctx, "Finish of DBaaS backup deleting")

	return diag.Diagnostics{}
}

func resourceDBaaSBackupImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	// Import format: project_id:region_id:backup_id.
	projectID, regionID, backupID, err := ImportStringParser(d.Id())
	if err != nil {
		return nil, fmt.Errorf("importing DBaaS backup: %w", err)
	}
	if err := d.Set(ProjectIDField, projectID); err != nil {
		return nil, fmt.Errorf("setting project_id: %w", err)
	}
	if err := d.Set(RegionIDField, regionID); err != nil {
		return nil, fmt.Errorf("setting region_id: %w", err)
	}
	d.SetId(backupID)

	return []*schema.ResourceData{d}, nil
}
