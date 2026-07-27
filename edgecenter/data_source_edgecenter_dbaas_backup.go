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

const dbaasBackupListPageSize = 100

func dataSourceDBaaSBackup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDBaaSBackupRead,
		Description: "Retrieve a DBaaS backup by its ID or unique name.",
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type: schema.TypeInt, Optional: true,
				Description:  "The project ID. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The project name. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type: schema.TypeInt, Optional: true,
				Description:  "The region ID. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The region name. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			IDField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The backup UUID. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{IDField, NameField},
				AtLeastOneOf: []string{IDField, NameField},
			},
			NameField: {
				Type: schema.TypeString, Optional: true,
				Description:  "The unique backup name. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{IDField, NameField},
				AtLeastOneOf: []string{IDField, NameField},
			},
			DescriptionField:              computedStringSchema(),
			DBaaSClusterIDField:           computedStringSchema(),
			DBaaSBackupParentIDField:      computedStringSchema(),
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

func computedStringSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Computed: true}
}

func dataSourceDBaaSBackupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	backupID, ok := d.GetOk(IDField)
	if !ok || backupID.(string) == "" {
		name := d.Get(NameField).(string)
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

// setDBaaSBackupData is shared by the data source and the future resource Read.
func setDBaaSBackupData(d *schema.ResourceData, client *edgecloudV2.Client, backup *edgecloudV2.DBaaSBackup) error {
	values := map[string]interface{}{
		ProjectIDField:                client.Project,
		RegionIDField:                 client.Region,
		NameField:                     backup.Name,
		DescriptionField:              backup.Description,
		DBaaSClusterIDField:           backup.ClusterID,
		DBaaSBackupParentIDField:      backup.ParentID,
		DBaaSBackupTypeField:          backup.BackupType,
		StatusField:                   backup.Status,
		DBaaSBackupSizeField:          backup.Size,
		DBaaSBackupIsServiceField:     backup.IsService,
		DBaaSBackupHasChildField:      backup.HasChild,
		CreatedAtField:                backup.CreatedAt,
		UpdatedAtField:                backup.UpdatedAt,
		DBaaSBackupFinishedAtField:    backup.FinishedAt,
		DBaaSClusterTaskIDField:       backup.TaskID,
		DBaaSBackupCreatorTaskIDField: backup.CreatorTaskID,
	}
	for field, value := range values {
		if err := d.Set(field, value); err != nil {
			return fmt.Errorf("setting DBaaS backup %s: %w", field, err)
		}
	}

	var dbms []interface{}
	if backup.DBMS != nil {
		dbms = []interface{}{map[string]interface{}{
			TypeField:             backup.DBMS.Type,
			DBaaSDbmsVersionField: backup.DBMS.Version,
		}}
	}
	if err := d.Set("dbms", dbms); err != nil {
		return fmt.Errorf("setting DBaaS backup dbms: %w", err)
	}

	return nil
}
