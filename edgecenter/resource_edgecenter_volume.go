package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
)

const (
	volumeDeleting        int = 1200
	VolumeCreatingTimeout int = 1200
	volumeExtending       int = 1200
	VolumesPoint              = "volumes"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVolumeCreate,
		ReadContext:   resourceVolumeRead,
		UpdateContext: resourceVolumeUpdate,
		DeleteContext: resourceVolumeDelete,
		Description: `A volume is a detachable block storage device akin to a USB hard drive or SSD, but located remotely in the cloud.
Volumes can be attached to a virtual machine and manipulated like a physical hard drive.`,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, volumeID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(volumeID)

				config := meta.(*Config)
				provider := config.Provider

				client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
				if err != nil {
					return nil, err
				}

				volume, err := volumes.Get(client, volumeID).Extract()
				if err != nil {
					return nil, fmt.Errorf("cannot get volume with ID: %s. Error: %w", volumeID, err)
				}
				d.Set("image_id", volume.VolumeImageMetadata.ImageID)

				return []*schema.ResourceData{d}, nil
			},
		},

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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the volume.",
			},
			"size": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The size of the volume, specified in gigabytes (GB).",
			},
			"type_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The type of volume to create. Valid values are 'ssd_hiiops', 'standard', 'cold', and 'ultra'. Defaults to 'standard'.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "(ForceNew) The ID of the image to create the volume from. This field is mandatory if creating a volume from an image.",
			},
			"snapshot_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "(ForceNew) The ID of the snapshot to create the volume from. This field is mandatory if creating a volume from a snapshot.",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
			"metadata_map": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceVolumeCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	opts, err := getVolumeData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	results, err := volumes.Create(client, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	VolumeID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, VolumeCreatingTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		volumeID, err := volumes.ExtractVolumeIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve volume ID from task info: %w", err)
		}
		return volumeID, nil
	},
	)
	log.Printf("[DEBUG] Volume id (%s)", VolumeID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(VolumeID.(string))
	resourceVolumeRead(ctx, d, m)

	log.Printf("[DEBUG] Finish volume creating (%s)", VolumeID)

	return diags
}

func resourceVolumeRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume reading")
	log.Printf("[DEBUG] Start volume reading%s", d.State())
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider
	volumeID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", volumeID)

	client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	volume, err := volumes.Get(client, volumeID).Extract()
	if err != nil {
		return diag.Errorf("cannot get volume with ID: %s. Error: %s", volumeID, err)
	}

	d.Set("name", volume.Name)
	d.Set("size", volume.Size)
	d.Set("type_name", volume.VolumeType)
	d.Set("region_id", volume.RegionID)
	d.Set("project_id", volume.ProjectID)

	metadataMap, metadataReadOnly := PrepareMetadata(volume.Metadata)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	fields := []string{"image_id", "snapshot_id"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish volume reading")

	return diags
}

func resourceVolumeUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume updating")
	volumeID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", volumeID)
	config := m.(*Config)
	provider := config.Provider
	client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	volume, err := volumes.Get(client, volumeID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		_, err := volumes.Update(client, volumeID, volumes.UpdateOpts{Name: name}).Extract()
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("size") {
		newValue := d.Get("size")
		newSize := newValue.(int)
		if newSize != 0 {
			if volume.Size < newSize {
				err = ExtendVolume(client, volumeID, newSize)
				if err != nil {
					return diag.FromErr(err)
				}
			} else {
				return diag.Errorf("Validation error: unable to update size field because new volume size must be greater than current size")
			}
		}
	}

	if d.HasChange("type_name") {
		newTN := d.Get("type_name")
		newVolumeType, err := volumes.VolumeType(newTN.(string)).ValidOrNil()
		if err != nil {
			return diag.FromErr(err)
		}

		opts := volumes.VolumeTypePropertyOperationOpts{
			VolumeType: *newVolumeType,
		}
		_, err = volumes.Retype(client, volumeID, opts).Extract()
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := utils.MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		err = metadata.ResourceMetadataReplace(client, d.Id(), meta).Err
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish volume updating")

	return resourceVolumeRead(ctx, d, m)
}

func resourceVolumeDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider
	volumeID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", volumeID)

	client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := volumes.DeleteOpts{
		Snapshots: [](string){d.Get("snapshot_id").(string)},
	}
	results, err := volumes.Delete(client, volumeID, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, volumeDeleting, func(task tasks.TaskID) (interface{}, error) {
		_, err := volumes.Get(client, volumeID).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete volume with ID: %s", volumeID)
		}
		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil, nil
		}
		return nil, fmt.Errorf("extracting Volume resource error: %w", err)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of volume deleting")

	return diags
}

func getVolumeData(d *schema.ResourceData) (*volumes.CreateOpts, error) {
	volumeData := volumes.CreateOpts{}
	volumeData.Source = volumes.NewVolume
	volumeData.Name = d.Get("name").(string)
	volumeData.Size = d.Get("size").(int)

	imageID := d.Get("image_id").(string)
	if imageID != "" {
		volumeData.Source = volumes.Image
		volumeData.ImageID = imageID
	}

	snapshotID := d.Get("snapshot_id").(string)
	if snapshotID != "" {
		volumeData.Source = volumes.Snapshot
		volumeData.SnapshotID = snapshotID
		if volumeData.Size != 0 {
			log.Println("[DEBUG] Size must be equal or larger to respective snapshot size")
		}
	}

	typeName := d.Get("type_name").(string)
	if typeName != "" {
		modifiedTypeName, err := volumes.VolumeType(typeName).ValidOrNil()
		if err != nil {
			return nil, fmt.Errorf("checking Volume validation error: %w", err)
		}
		volumeData.TypeName = *modifiedTypeName
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := utils.MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("volume metadata error: %w", err)
		}

		volumeData.Metadata = meta
	}

	return &volumeData, nil
}

func ExtendVolume(client *edgecloud.ServiceClient, volumeID string, newSize int) error {
	opts := volumes.SizePropertyOperationOpts{
		Size: newSize,
	}
	results, err := volumes.Extend(client, volumeID, opts).Extract()
	if err != nil {
		return fmt.Errorf("extracting Volume resource error: %w", err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, volumeExtending, func(task tasks.TaskID) (interface{}, error) {
		_, err := volumes.Get(client, volumeID).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get volume with ID: %s. Error: %w", volumeID, err)
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("checking Volume state error: %w", err)
	}
	log.Printf("[DEBUG] Finish waiting.")

	return nil
}
