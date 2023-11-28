package volume

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func ResourceEdgeCenterVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterVolumeCreate,
		ReadContext:   resourceEdgeCenterVolumeRead,
		UpdateContext: resourceEdgeCenterVolumeUpdate,
		DeleteContext: resourceEdgeCenterVolumeDelete,
		Description: `A volume is a detachable block storage device akin to a USB hard drive or SSD, but located remotely in the cloud.
Volumes can be attached to a virtual machine and manipulated like a physical hard drive.`,
		Schema: volumeSchema(),

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
			// if the new size of the volume is smaller than the old one return an error since
			// only expanding the volume is allowed
			oldSize, newSize := diff.GetChange("size")
			if newSize.(int) < oldSize.(int) {
				return fmt.Errorf("volumes `size` can only be expanded and not shrunk")
			}

			return nil
		},
	}
}

func resourceEdgeCenterVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.VolumeCreateRequest{
		Name:     d.Get("name").(string),
		Size:     d.Get("size").(int),
		TypeName: edgecloud.VolumeType(d.Get("volume_type").(string)),
	}

	if v, ok := d.GetOk("metadata"); ok {
		metadata := converter.MapInterfaceToMapString(v.(map[string]interface{}))
		opts.Metadata = metadata
	}

	source := d.Get("source").(string)
	opts.Source = edgecloud.VolumeSource(source)
	switch source {
	case "snapshot":
		if v, ok := d.GetOk("snapshot_id"); ok {
			opts.SnapshotID = v.(string)
		} else {
			return diag.Errorf("'snapshot_id' is mandatory if creating a volume from an image")
		}
	case "image":
		if v, ok := d.GetOk("image_id"); ok {
			opts.ImageID = v.(string)
		} else {
			return diag.Errorf("'image_id' is mandatory if creating a volume from an image")
		}
	}

	if v, ok := d.GetOk("instance_id_to_attach_to"); ok {
		opts.InstanceIDToAttachTo = v.(string)
		if attachmentTag, okTag := d.GetOk("attachment_tag"); okTag {
			opts.AttachmentTag = attachmentTag.(string)
		}
	}

	log.Printf("[DEBUG] Volume create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Volumes.Create, opts, client)
	if err != nil {
		return diag.Errorf("error creating volume: %s", err)
	}

	d.SetId(taskResult.Volumes[0])

	log.Printf("[INFO] Volume: %s", d.Id())

	return resourceEdgeCenterVolumeRead(ctx, d, meta)
}

func resourceEdgeCenterVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the volume properties for updating the state
	foundVolume, resp, err := client.Volumes.Get(ctx, d.Id())
	if err != nil {
		// check if volume no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter Volume (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving volume: %s", err)
	}

	d.Set("volume_type", foundVolume.VolumeType)
	d.Set("region", foundVolume.Region)
	d.Set("status", foundVolume.Status)
	d.Set("bootable", foundVolume.Bootable)
	d.Set("limiter_stats", foundVolume.LimiterStats)
	d.Set("snapshot_ids", foundVolume.SnapshotIDs)
	d.Set("limiter_stats",
		map[string]int{
			"iops_base_limit":  foundVolume.LimiterStats.IopsBaseLimit,
			"iops_burst_limit": foundVolume.LimiterStats.IopsBurstLimit,
			"MBps_base_limit":  foundVolume.LimiterStats.MBpsBaseLimit,
			"MBps_burst_limit": foundVolume.LimiterStats.MBpsBurstLimit,
		})
	if len(foundVolume.Attachments) > 0 {
		attachments := make([]map[string]interface{}, 0, len(foundVolume.Attachments))
		for _, attachment := range foundVolume.Attachments {
			attachments = append(attachments, map[string]interface{}{
				"volume_id":     attachment.VolumeID,
				"attachment_id": attachment.AttachmentID,
				"server_id":     attachment.ServerID,
			})
		}
		d.Set("attachments", attachments)
	}

	return nil
}

func resourceEdgeCenterVolumeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		_, _, err := client.Volumes.Rename(ctx, d.Id(), &edgecloud.Name{Name: newName})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("instance_id_to_attach_to") {
		oldInstance, newInstance := d.GetChange("instance_id_to_attach_to")

		if oldInstance != "" {
			volumeDetachRequest := &edgecloud.VolumeDetachRequest{InstanceID: oldInstance.(string)}
			if _, _, err := client.Volumes.Detach(ctx, d.Id(), volumeDetachRequest); err != nil {
				return diag.Errorf("Error detaching volume from instance: %s", err)
			}
		}

		if newInstance != "" {
			volumeAttachRequest := &edgecloud.VolumeAttachRequest{
				InstanceID:    newInstance.(string),
				AttachmentTag: d.Get("attachment_tag").(string),
			}
			if _, _, err := client.Volumes.Attach(ctx, d.Id(), volumeAttachRequest); err != nil {
				return diag.Errorf("Error attaching volume to instance: %s", err)
			}
		}
	}

	if d.HasChange("size") {
		newSize := d.Get("size").(int)
		task, _, err := client.Volumes.Extend(ctx, d.Id(), &edgecloud.VolumeExtendSizeRequest{Size: newSize})
		if err != nil {
			return diag.FromErr(err)
		}
		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("volume_type") {
		newVolumeType := d.Get("volume_type").(string)
		_, _, err := client.Volumes.ChangeType(ctx, d.Id(), &edgecloud.VolumeChangeTypeRequest{
			VolumeType: edgecloud.VolumeType(newVolumeType),
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("metadata") {
		metadata := edgecloud.Metadata(converter.MapInterfaceToMapString(d.Get("metadata").(map[string]interface{})))

		_, err := client.Volumes.MetadataUpdate(ctx, d.Id(), &metadata)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	return resourceEdgeCenterVolumeRead(ctx, d, meta)
}

func resourceEdgeCenterVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	volume, _, err := client.Volumes.Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("Error getting volume: %s", err)
	}

	if len(volume.Attachments) > 0 {
		volumeDetachRequest := &edgecloud.VolumeDetachRequest{InstanceID: volume.Attachments[0].ServerID}
		if _, _, err = client.Volumes.Detach(ctx, d.Id(), volumeDetachRequest); err != nil {
			return diag.Errorf("Error detaching volume from instance: %s", err)
		}
	}

	log.Printf("[INFO] Deleting volume: %s", d.Id())
	if err := util.DeleteResourceIfExist(ctx, client, client.Volumes, d.Id()); err != nil {
		return diag.Errorf("Error deleting volume: %s", err)
	}
	d.SetId("")

	return nil
}
