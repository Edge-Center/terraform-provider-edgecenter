package instance

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"slices"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func ResourceEdgeCenterInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterInstanceCreate,
		ReadContext:   resourceEdgeCenterInstanceRead,
		UpdateContext: resourceEdgeCenterInstanceUpdate,
		DeleteContext: resourceEdgeCenterInstanceDelete,
		Description:   `A cloud instance is a virtual machine in a cloud environment`,
		Schema:        instanceSchema(),

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
			oldVolumesRaw, newVolumesRaw := diff.GetChange("volume")
			oldVolumes, newVolumes := oldVolumesRaw.([]interface{}), newVolumesRaw.([]interface{})

			newVolumesBootIndexes := getVolumesBootIndexList(newVolumes)

			if !slices.Contains(newVolumesBootIndexes, 0) {
				return fmt.Errorf("one of volumes should be with boot_index = 0")
			}

			// sequential means 0, 1, 2, 3 but not 0, 2, 3, 1
			if len(newVolumesBootIndexes) > 1 {
				for i := 1; i < len(newVolumesBootIndexes); i++ {
					if newVolumesBootIndexes[i]-newVolumesBootIndexes[i-1] != 1 {
						return fmt.Errorf("volume boot_index order must be sequential")
					}
				}
			}

			// check same volume changed
			for _, v := range newVolumes {
				volume := v.(map[string]interface{})
				oldVolumeWithSameID := getVolumeInfoByID(volume["id"].(string), oldVolumes)

				if oldVolumeWithSameID != nil {
					if oldVolumeWithSameID["size"].(int) > volume["size"].(int) {
						return fmt.Errorf("volumes `size` can only be expanded and not shrunk")
					}

					if oldVolumeWithSameID["name"].(string) != volume["name"].(string) {
						return fmt.Errorf("volume cannot be renamed. create a new one with the name you want")
					}

					if oldVolumeWithSameID["type_name"].(string) != volume["type_name"].(string) {
						return fmt.Errorf("volume type cannot changed. create a new one with the type you want")
					}
				}
			}

			return nil
		},
	}
}

func resourceEdgeCenterInstanceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.InstanceCreateRequest{
		Flavor:        d.Get("flavor").(string),
		KeypairName:   d.Get("keypair_name").(string),
		ServerGroupID: d.Get("server_group_id").(string),
		Username:      d.Get("username").(string),
		Password:      d.Get("password").(string),
		AllowAppPorts: d.Get("allow_app_ports").(bool),
	}

	if userData, ok := d.GetOk("user_data"); ok {
		opts.UserData = base64.StdEncoding.EncodeToString([]byte(userData.(string)))
	}

	if v, ok := d.GetOk("name"); ok {
		opts.Names = []string{v.(string)}
	} else if v, ok := d.GetOk("name_templates"); ok {
		nameTemplates := v.([]string)
		opts.NameTemplates = nameTemplates
	}

	if v, ok := d.GetOk("security_groups"); ok {
		securityGroups := v.([]interface{})
		sgsList := make([]edgecloud.ID, 0, len(securityGroups))
		for _, sg := range securityGroups {
			sgsList = append(sgsList, edgecloud.ID{ID: sg.(string)})
		}
		opts.SecurityGroups = sgsList
	}

	volumes := d.Get("volume").([]interface{})
	instanceVolumeCreateList, err := converter.ListInterfaceToListInstanceVolumeCreate(volumes)
	if err != nil {
		return diag.Errorf("error creating volume config: %s", err)
	}
	opts.Volumes = instanceVolumeCreateList

	if v, ok := d.GetOk("metadata"); ok {
		metadata := converter.MapInterfaceToMapString(v.(map[string]interface{}))
		opts.Metadata = metadata
	}

	ifs := d.Get("interface").([]interface{})
	interfaceInstanceCreateOptsList, err := converter.ListInterfaceToListInstanceInterface(ifs)
	if err != nil {
		return diag.Errorf("error creating interface config: %s", err)
	}

	if v, ok := d.GetOk("security_groups"); ok {
		securityGroups := v.([]interface{})
		sgsList := make([]edgecloud.ID, 0, len(securityGroups))
		for _, sg := range securityGroups {
			sgsList = append(sgsList, edgecloud.ID{ID: sg.(string)})
		}
		opts.SecurityGroups = sgsList
	}

	opts.Interfaces = interfaceInstanceCreateOptsList

	log.Printf("[DEBUG] Instance create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Instances.Create, opts, client)
	if err != nil {
		return diag.Errorf("error creating volume: %s", err)
	}

	d.SetId(taskResult.Instances[0])

	log.Printf("[INFO] Instance: %s", d.Id())

	return resourceEdgeCenterInstanceRead(ctx, d, meta)
}

func resourceEdgeCenterInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the volume properties for updating the state
	foundInstance, resp, err := client.Instances.Get(ctx, d.Id())
	if err != nil {
		// check if instance no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter Instance (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving instance: %s", err)
	}

	d.Set("status", foundInstance.Status)
	d.Set("region", foundInstance.Region)
	d.Set("vm_state", foundInstance.VMState)
	d.Set("keypair_id", foundInstance.KeypairName)

	// desc sorting: first volume is the last added
	volumes, _, err := client.Volumes.List(ctx, &edgecloud.VolumeListOptions{InstanceID: d.Id()})
	if err != nil {
		return diag.Errorf("Error retrieving instance volumes: %s", err)
	}

	getID := func(name string, volumeList []edgecloud.Volume) string {
		for _, volume := range volumeList {
			if volume.Name == name {
				return volume.ID
			}
		}

		return ""
	}

	// asc sorting: first volume is the first added
	currentVolumes := d.Get("volume").([]interface{})
	for i, v := range currentVolumes {
		volume := v.(map[string]interface{})
		volumeID := getID(volume["name"].(string), volumes)
		if volumeID == "" {
			return diag.Errorf("Error during get volume id")
		}
		currentVolumes[i].(map[string]interface{})["id"] = volumeID
	}

	if err := d.Set("volume", currentVolumes); err != nil {
		return diag.FromErr(err)
	}

	if len(foundInstance.MetadataDetailed) > 0 {
		metadata := make([]map[string]interface{}, 0, len(foundInstance.MetadataDetailed))
		for _, metadataItem := range foundInstance.MetadataDetailed {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
		d.Set("metadata_detailed", metadata)
	}

	addresses := make([]map[string]string, 0, len(foundInstance.Addresses))
	for networkName, networkInfo := range foundInstance.Addresses {
		net := networkInfo[0]
		address := map[string]string{
			"network_name": networkName,
			"type":         net.Type,
			"addr":         net.Address.String(),
			"subnet_id":    net.SubnetID,
			"subnet_name":  net.SubnetName,
		}
		addresses = append(addresses, address)
	}
	if err := d.Set("addresses", addresses); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceEdgeCenterInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics { //nolint: gocognit
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		if _, _, err := client.Instances.Rename(ctx, d.Id(), &edgecloud.Name{Name: newName}); err != nil {
			return diag.Errorf("Error when renaming the instance: %s", err)
		}
	}

	if d.HasChange("flavor") {
		newFlavor := d.Get("flavor").(string)
		instanceFlavorUpdateRequest := &edgecloud.InstanceFlavorUpdateRequest{FlavorID: newFlavor}
		task, _, err := client.Instances.UpdateFlavor(ctx, d.Id(), instanceFlavorUpdateRequest)
		if err != nil {
			return diag.Errorf("Error when changing the instance flavor: %s", err)
		}

		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return diag.Errorf("Error while waiting for flavor change: %s", err)
		}
	}

	if d.HasChange("metadata") {
		metadata := edgecloud.Metadata(converter.MapInterfaceToMapString(d.Get("metadata").(map[string]interface{})))

		if _, err := client.Instances.MetadataUpdate(ctx, d.Id(), &metadata); err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	if d.HasChange("server_group_id") {
		oldSgRaw, newSgRaw := d.GetChange("server_group_id")
		oldSg, newSg := oldSgRaw.(string), newSgRaw.(string)

		// delete old server group
		if oldSg != "" {
			task, _, err := client.Instances.RemoveFromServerGroup(ctx, d.Id())
			if err != nil {
				return diag.Errorf("Error when remove the instance from server group: %s", err)
			}

			if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
				return diag.Errorf("Error while waiting for instance remove from server group: %s", err)
			}
		}

		// add new server group if needed
		if newSg != "" {
			instancePutIntoServerGroupRequest := &edgecloud.InstancePutIntoServerGroupRequest{ServerGroupID: newSg}
			task, _, err := client.Instances.PutIntoServerGroup(ctx, d.Id(), instancePutIntoServerGroupRequest)
			if err != nil {
				return diag.Errorf("Error when put the instance to new server group: %s", err)
			}

			if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
				return diag.Errorf("Error while waiting for instance put to new server group: %s", err)
			}
		}
	}

	if d.HasChange("volume") {
		oldVolumesRaw, newVolumesRaw := d.GetChange("volume")
		oldVolumes, newVolumes := oldVolumesRaw.([]interface{}), newVolumesRaw.([]interface{})

		oldIDs := getVolumeIDsSet(oldVolumes)
		newIDs := getVolumeIDsSet(newVolumes)

		// detach volumes
		for volumeID := range converter.MapLeftDiff(oldIDs, newIDs) {
			volume := getVolumeInfoByID(volumeID, oldVolumes)
			if volume["boot_index"].(int) == 0 {
				return diag.Errorf("cannot detach primary boot device with boot_index=0. id: %s", volumeID)
			}

			volumeDetachRequest := &edgecloud.VolumeDetachRequest{InstanceID: d.Id()}
			if _, _, err := client.Volumes.Detach(ctx, volumeID, volumeDetachRequest); err != nil {
				return diag.Errorf("Error while detaching the volume: %s", err)
			}
		}

		// attach volumes
		for volumeID := range converter.MapLeftDiff(newIDs, oldIDs) {
			volume := getVolumeInfoByID(volumeID, newVolumes)
			attachmentTag := volume["attachment_tag"].(string)

			switch volume["source"].(string) {
			case "image":
				return diag.Errorf("cannot attach image-source volume, required 'existing-volume' or 'new-volume' source")
			case "existing-volume":
				volumeAttachRequest := &edgecloud.VolumeAttachRequest{
					InstanceID:    d.Id(),
					AttachmentTag: attachmentTag,
				}
				if _, _, err := client.Volumes.Attach(ctx, volume["volume_id"].(string), volumeAttachRequest); err != nil {
					return diag.Errorf("cannot attach existing-volume: %s. Error: %s", volumeID, err)
				}
			case "new-volume":
				volumeCreateRequest := &edgecloud.VolumeCreateRequest{
					AttachmentTag:        attachmentTag,
					Source:               "new-volume",
					InstanceIDToAttachTo: d.Id(),
					Name:                 volume["name"].(string),
					Size:                 volume["size"].(int),
					TypeName:             edgecloud.VolumeType(volume["type_name"].(string)),
				}
				task, _, err := client.Volumes.Create(ctx, volumeCreateRequest)
				if err != nil {
					return diag.Errorf("Error when creating a new instance volume: %s", err)
				}
				if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
					return diag.Errorf("Error while waiting for instance volume create: %s", err)
				}
			}
		}

		// resize the same volume
		for volumeID := range converter.MapsIntersection(newIDs, oldIDs) {
			volumeOld := getVolumeInfoByID(volumeID, oldVolumes)
			volumeNew := getVolumeInfoByID(volumeID, newVolumes)

			if volumeOld["size"].(int) != volumeNew["size"].(int) {
				volumeExtendSizeRequest := &edgecloud.VolumeExtendSizeRequest{Size: volumeNew["size"].(int)}
				task, _, err := client.Volumes.Extend(ctx, volumeID, volumeExtendSizeRequest)
				if err != nil {
					return diag.FromErr(err)
				}
				if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	return resourceEdgeCenterInstanceRead(ctx, d, meta)
}

func resourceEdgeCenterInstanceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting instance: %s", d.Id())
	task, _, err := client.Instances.Delete(ctx, d.Id(), nil)
	if err != nil {
		return diag.Errorf("Error deleting instance: %s", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Delete instance task failed with error: %s", err)
	}

	if err = util.ResourceIsDeleted(ctx, client.Instances.Get, d.Id()); err != nil {
		return diag.Errorf("Instance with id %s was not deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func getVolumeIDsSet(volumes []interface{}) map[string]struct{} {
	ids := make(map[string]struct{}, len(volumes))
	for _, volumeRaw := range volumes {
		volume := volumeRaw.(map[string]interface{})
		ids[volume["id"].(string)] = struct{}{}
	}

	return ids
}

func getVolumeInfoByID(id string, volumeList []interface{}) map[string]interface{} {
	for _, volumeRaw := range volumeList {
		volume := volumeRaw.(map[string]interface{})
		if volume["id"].(string) == id {
			return volume
		}
	}

	return nil
}

func getVolumesBootIndexList(volumes []interface{}) []int {
	idxList := make([]int, 0, len(volumes))
	for _, volumeRaw := range volumes {
		volume := volumeRaw.(map[string]interface{})
		idxList = append(idxList, volume["boot_index"].(int))
	}

	return idxList
}
