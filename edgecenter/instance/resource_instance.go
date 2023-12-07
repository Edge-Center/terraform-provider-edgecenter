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
		return diag.Errorf("error creating instance volume config: %s", err)
	}
	opts.Volumes = instanceVolumeCreateList

	if v, ok := d.GetOk("metadata"); ok {
		metadata := converter.MapInterfaceToMapString(v.(map[string]interface{}))
		opts.Metadata = metadata
	}

	ifs := d.Get("interface").([]interface{})
	interfaceInstanceCreateOptsList, err := converter.ListInterfaceToListInstanceInterface(ifs)
	if err != nil {
		return diag.Errorf("error creating instance interface config: %s", err)
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
		return diag.Errorf("error creating instance: %s", err)
	}

	d.SetId(taskResult.Instances[0])

	log.Printf("[INFO] Instance: %s", d.Id())

	return resourceEdgeCenterInstanceRead(ctx, d, meta)
}

func resourceEdgeCenterInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the instance properties for updating the state
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

	if err = setVolumes(ctx, d, client); err != nil {
		return diag.FromErr(err)
	}

	if err = setInterfaces(ctx, d, client); err != nil {
		return diag.FromErr(err)
	}

	if err = setAddresses(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
	}

	if err = setMetadataDetailed(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceEdgeCenterInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics { //nolint: gocognit, gocyclo
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
		if err := changeServerGroup(ctx, d, client); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("volume") {
		if err := changeVolumes(ctx, d, client); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("interface") {
		iOldRaw, iNewRaw := d.GetChange("interface")
		ifsOldSlice, ifsNewSlice := iOldRaw.([]interface{}), iNewRaw.([]interface{})

		switch {
		// the same number of interfaces
		case len(ifsOldSlice) == len(ifsNewSlice):
			for idx, item := range ifsOldSlice {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDs(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDs(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					unAssignSGs := getSecurityGroupsDifference(sgsIDsNew, sgsIDsOld)
					assignSGs := getSecurityGroupsDifference(sgsIDsOld, sgsIDsNew)
					if err := updateInterfaceSecurityGroups(ctx, d, client, portID, unAssignSGs, assignSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := converter.MapDifference(iOld, iNew, []string{"security_groups"})
				if len(differentFields) > 0 {
					if err := detachInterface(ctx, d, client, iOld); err != nil {
						return diag.FromErr(err)
					}

					if err := attachInterface(ctx, d, client, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

		// new interfaces > old interfaces - need to attach new
		case len(ifsOldSlice) < len(ifsNewSlice):
			for idx, item := range ifsOldSlice {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDs(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDs(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					unAssignSGs := getSecurityGroupsDifference(sgsIDsNew, sgsIDsOld)
					assignSGs := getSecurityGroupsDifference(sgsIDsOld, sgsIDsNew)
					if err := updateInterfaceSecurityGroups(ctx, d, client, portID, unAssignSGs, assignSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := converter.MapDifference(iOld, iNew, []string{"security_groups"})
				if len(differentFields) > 0 {
					if err := detachInterface(ctx, d, client, iOld); err != nil {
						return diag.FromErr(err)
					}

					if err := attachInterface(ctx, d, client, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

			for _, item := range ifsNewSlice[len(ifsOldSlice):] {
				iNew := item.(map[string]interface{})
				if err := attachInterface(ctx, d, client, iNew); err != nil {
					return diag.FromErr(err)
				}
			}

			// old interfaces > new interfaces - need to detach old
		case len(ifsOldSlice) > len(ifsNewSlice):
			for idx, item := range ifsOldSlice[:len(ifsNewSlice)] {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDs(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDs(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					unAssignSGs := getSecurityGroupsDifference(sgsIDsNew, sgsIDsOld)
					assignSGs := getSecurityGroupsDifference(sgsIDsOld, sgsIDsNew)
					if err := updateInterfaceSecurityGroups(ctx, d, client, portID, unAssignSGs, assignSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := converter.MapDifference(iOld, iNew, []string{"security_groups"})
				if len(differentFields) > 0 {
					if err := detachInterface(ctx, d, client, iOld); err != nil {
						return diag.FromErr(err)
					}

					if err := attachInterface(ctx, d, client, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

			for _, item := range ifsOldSlice[len(ifsNewSlice):] {
				iOld := item.(map[string]interface{})
				if err := detachInterface(ctx, d, client, iOld); err != nil {
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
