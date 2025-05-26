package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceSnapshot() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSnapshotRead,
		Description: "A snapshot is a feature that allows you to capture the current state of the instance or volume at a specific point in time",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the snapshot. Either 'name' or 'snapshot_id' must be specified.",
				ExactlyOneOf: []string{"name", "snapshot_id"},
			},
			"snapshot_id": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The ID of the snapshot.Either 'name' or 'snapshot_id' must be specified.",
				ExactlyOneOf: []string{"name", "snapshot_id"},
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the snapshot.",
			},
			"creator_task_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The task that created this entity.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the snapshot.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the snapshot, GiB.",
			},
			"volume_id": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The ID of the volume this snapshot was made from.",
				RequiredWith: []string{"name"},
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the snapshot was last updated.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the snapshot was created.",
			},
			"metadata": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The metadata",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceSnapshotRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start snapshot reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	snapshot, err := getSnapshot(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	setSnapshotData(d, snapshot)

	log.Println("[DEBUG] Finish snapshot reading")

	return nil
}

func setSnapshotData(d *schema.ResourceData, snapshot *edgecloudV2.Snapshot) {
	d.SetId(snapshot.ID)
	_ = d.Set("name", snapshot.Name)
	_ = d.Set("updated_at", snapshot.UpdatedAt)
	_ = d.Set("created_at", snapshot.CreatedAt)
	_ = d.Set("status", snapshot.Status)
	_ = d.Set("creator_task_id", snapshot.CreatorTaskID)
	_ = d.Set("size", snapshot.Size)
	_ = d.Set("volume_id", snapshot.VolumeID)
	_ = d.Set("description", snapshot.Description)
	_ = d.Set("snapshot_id", snapshot.ID)
	_ = d.Set("metadata", snapshot.Metadata)
}
