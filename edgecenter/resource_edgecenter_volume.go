package edgecenter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	volumeDeletingTimeout  = 1200 * time.Second
	VolumeCreatingTimeout  = 1200 * time.Second
	volumeExtendingTimeout = 1200 * time.Second
	VolumesPoint           = "volumes"
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
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, volumeID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(volumeID)

				config := m.(*Config)
				clientV2, err := config.newCloudClient()
				if err != nil {
					return nil, err
				}

				clientV2.Region = regionID
				clientV2.Project = projectID

				volume, _, err := clientV2.Volumes.Get(ctx, volumeID)
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

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	opts, err := getVolumeDataV2(d)
	if err != nil {
		return diag.FromErr(err)
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Volumes.Create, opts, clientV2, VolumeCreatingTimeout)
	if err != nil {
		return diag.Errorf("error creating volume: %s", err)
	}

	VolumeID := taskResult.Volumes[0]

	log.Printf("[DEBUG] Volume id (%s)", VolumeID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(VolumeID)
	resourceVolumeRead(ctx, d, m)

	log.Printf("[DEBUG] Finish volume creating (%s)", VolumeID)

	return diags
}

func resourceVolumeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume reading")
	log.Printf("[DEBUG] Start volume reading%s", d.State())
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	volumeID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", volumeID)

	volume, _, err := clientV2.Volumes.Get(ctx, volumeID)
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

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	volume, _, err := clientV2.Volumes.Get(ctx, volumeID)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		_, _, err := clientV2.Volumes.Rename(ctx, volumeID, &edgecloudV2.Name{Name: newName})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("size") {
		newSize := d.Get("size").(int)
		if newSize != 0 {
			if volume.Size < newSize {
				task, _, err := clientV2.Volumes.Extend(ctx, volumeID, &edgecloudV2.VolumeExtendSizeRequest{Size: newSize})
				if err != nil {
					return diag.FromErr(err)
				}
				if err = utilV2.WaitForTaskComplete(ctx, clientV2, task.Tasks[0], volumeExtendingTimeout); err != nil {
					return diag.FromErr(err)
				}
			} else {
				return diag.Errorf("Validation error: unable to update size field because new volume size must be greater than current size")
			}
		}
	}

	if d.HasChange("type_name") {
		newTN := d.Get("type_name")
		newVolumeType, err := edgecloudV2.VolumeType(newTN.(string)).ValidOrNil()
		if err != nil {
			return diag.FromErr(err)
		}

		_, _, err = clientV2.Volumes.ChangeType(ctx, volumeID, &edgecloudV2.VolumeChangeTypeRequest{
			VolumeType: *newVolumeType,
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		metadata, err := MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}
		metadataUpdate := edgecloudV2.Metadata(*metadata)

		if _, err := clientV2.Volumes.MetadataUpdate(ctx, d.Id(), &metadataUpdate); err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish volume updating")

	return resourceVolumeRead(ctx, d, m)
}

func resourceVolumeDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start volume deleting")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	volumeID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", volumeID)

	volume, _, err := clientV2.Volumes.Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("Error getting volume: %s", err)
	}

	if len(volume.Attachments) > 0 {
		volumeDetachRequest := &edgecloudV2.VolumeDetachRequest{InstanceID: volume.Attachments[0].ServerID}
		if _, _, err = clientV2.Volumes.Detach(ctx, d.Id(), volumeDetachRequest); err != nil {
			return diag.Errorf("Error detaching volume from instance: %s", err)
		}
	}

	log.Printf("[INFO] Deleting volume: %s", d.Id())
	if err = utilV2.DeleteResourceIfExist(ctx, clientV2, clientV2.Volumes, d.Id(), volumeDeletingTimeout); err != nil {
		return diag.Errorf("Error deleting volume: %s", err)
	}
	d.SetId("")

	return nil
}

func getVolumeDataV2(d *schema.ResourceData) (*edgecloudV2.VolumeCreateRequest, error) {
	volumeData := edgecloudV2.VolumeCreateRequest{}
	volumeData.Source = edgecloudV2.VolumeSourceNewVolume
	volumeData.Name = d.Get("name").(string)
	volumeData.Size = d.Get("size").(int)

	imageID := d.Get("image_id").(string)
	if imageID != "" {
		volumeData.Source = edgecloudV2.VolumeSourceImage
		volumeData.ImageID = imageID
	}

	snapshotID := d.Get("snapshot_id").(string)
	if snapshotID != "" {
		volumeData.Source = edgecloudV2.VolumeSourceSnapshot
		volumeData.SnapshotID = snapshotID
		if volumeData.Size != 0 {
			log.Println("[DEBUG] Size must be equal or larger to respective snapshot size")
		}
	}

	typeName := d.Get("type_name").(string)
	if typeName != "" {
		modifiedTypeName, err := edgecloudV2.VolumeType(typeName).ValidOrNil()
		if err != nil {
			return nil, fmt.Errorf("checking Volume validation error: %w", err)
		}
		volumeData.TypeName = *modifiedTypeName
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("volume metadata error: %w", err)
		}

		volumeData.Metadata = *meta
	}

	return &volumeData, nil
}
