package dbaas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const DBaaSBackupDataSource = "edgecenter_dbaas_backup"

const dbaasBackupListPageSize = 100

func dataSourceDBaaSBackup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSBackupRead,
		Description: "Retrieve a DBaaS backup by its ID or unique name.",
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type: schema.TypeInt, Optional: true,
				Description:  "The project ID. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The project name. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type: schema.TypeInt, Optional: true,
				Description:  "The region ID. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The region name. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.IDField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The backup UUID. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
				AtLeastOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},
			edgecenter.NameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The unique backup name. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
				AtLeastOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},
			edgecenter.DescriptionField:              computedStringSchema(),
			edgecenter.DBaaSClusterIDField:           computedStringSchema(),
			edgecenter.DBaaSBackupParentIDField:      computedStringSchema(),
			edgecenter.DBaaSBackupTypeField:          computedStringSchema(),
			edgecenter.StatusField:                   computedStringSchema(),
			edgecenter.DBaaSBackupSizeField:          {Type: schema.TypeFloat, Computed: true},
			edgecenter.DBaaSBackupIsServiceField:     {Type: schema.TypeBool, Computed: true},
			edgecenter.DBaaSBackupHasChildField:      {Type: schema.TypeBool, Computed: true},
			edgecenter.CreatedAtField:                computedStringSchema(),
			edgecenter.UpdatedAtField:                computedStringSchema(),
			edgecenter.DBaaSBackupFinishedAtField:    computedStringSchema(),
			edgecenter.DBaaSClusterTaskIDField:       computedStringSchema(),
			edgecenter.DBaaSBackupCreatorTaskIDField: computedStringSchema(),
			"dbms": {
				Type: schema.TypeList, Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					edgecenter.TypeField:             computedStringSchema(),
					edgecenter.DBaaSDbmsVersionField: computedStringSchema(),
				}},
			},
		},
	}
}

func computedStringSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Computed: true}
}

func dataSourceDBaaSBackupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	backupID, ok := d.GetOk(edgecenter.IDField)
	if !ok || backupID.(string) == "" {
		name := d.Get(edgecenter.NameField).(string)
		backups, err := findDBaaSBackupsByName(ctx, clientV2, name)
		if err != nil {
			return diag.FromErr(err)
		}

		var matches []edgecloudV2.DBaaSBackup
		for _, backup := range backups {
			if backup.Name == name {
				matches = append(matches, backup)
			}
		}
		if len(matches) == 0 {
			return diag.Errorf("DBaaS backup with name %q was not found", name)
		}
		if len(matches) > 1 {
			return diag.Errorf("multiple DBaaS backups have name %q; specify 'id' instead", name)
		}
		backupID = matches[0].ID
	}

	backup, resp, err := clientV2.DBaaS.BackupGet(ctx, backupID.(string), false)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return diag.Errorf("DBaaS backup %q was not found", backupID.(string))
		}
		return diag.FromErr(err)
	}
	if backup == nil {
		return diag.Errorf("DBaaS backup %q: empty API response", backupID.(string))
	}

	if err := setDBaaSBackupData(d, clientV2, backup); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(backup.ID)
	tflog.Debug(ctx, fmt.Sprintf("DBaaS backup data source read, ID: %s", backup.ID))

	return nil
}

func findDBaaSBackupsByName(ctx context.Context, client *edgecloudV2.Client, name string) ([]edgecloudV2.DBaaSBackup, error) {
	var backups []edgecloudV2.DBaaSBackup

	for offset := 0; ; {
		page, _, err := client.DBaaS.BackupsListPage(ctx, &edgecloudV2.DBaaSBackupListOptions{
			Search: name,
			Limit:  dbaasBackupListPageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing DBaaS backups page: %w", err)
		}

		backups = append(backups, page.Results...)
		if len(page.Results) == 0 || len(page.Results) < dbaasBackupListPageSize || (page.Count > 0 && offset+len(page.Results) >= page.Count) {
			return backups, nil
		}
		offset += len(page.Results)
	}
}

func setDBaaSBackupData(d *schema.ResourceData, client *edgecloudV2.Client, backup *edgecloudV2.DBaaSBackup) error {
	values := map[string]interface{}{
		edgecenter.ProjectIDField:                client.Project,
		edgecenter.RegionIDField:                 client.Region,
		edgecenter.NameField:                     backup.Name,
		edgecenter.DescriptionField:              backup.Description,
		edgecenter.DBaaSClusterIDField:           backup.ClusterID,
		edgecenter.DBaaSBackupParentIDField:      backup.ParentID,
		edgecenter.DBaaSBackupTypeField:          backup.BackupType,
		edgecenter.StatusField:                   backup.Status,
		edgecenter.DBaaSBackupSizeField:          backup.Size,
		edgecenter.DBaaSBackupIsServiceField:     backup.IsService,
		edgecenter.DBaaSBackupHasChildField:      backup.HasChild,
		edgecenter.CreatedAtField:                backup.CreatedAt,
		edgecenter.UpdatedAtField:                backup.UpdatedAt,
		edgecenter.DBaaSBackupFinishedAtField:    backup.FinishedAt,
		edgecenter.DBaaSClusterTaskIDField:       backup.TaskID,
		edgecenter.DBaaSBackupCreatorTaskIDField: backup.CreatorTaskID,
	}
	for field, value := range values {
		if err := d.Set(field, value); err != nil {
			return fmt.Errorf("setting DBaaS backup %s: %w", field, err)
		}
	}

	var dbms []interface{}
	if backup.DBMS != nil {
		dbms = []interface{}{map[string]interface{}{
			edgecenter.TypeField:             backup.DBMS.Type,
			edgecenter.DBaaSDbmsVersionField: backup.DBMS.Version,
		}}
	}
	if err := d.Set("dbms", dbms); err != nil {
		return fmt.Errorf("setting DBaaS backup dbms: %w", err)
	}

	return nil
}
