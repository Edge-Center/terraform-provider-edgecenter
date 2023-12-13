package instance

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func changeServerGroup(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	oldSgRaw, newSgRaw := d.GetChange("server_group_id")
	oldSg, newSg := oldSgRaw.(string), newSgRaw.(string)

	// delete old server group
	if oldSg != "" {
		task, _, err := client.Instances.RemoveFromServerGroup(ctx, d.Id())
		if err != nil {
			return fmt.Errorf("error when remove the instance from server group: %w", err)
		}

		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return fmt.Errorf("error while waiting for instance remove from server group: %w", err)
		}
	}

	// add new server group if needed
	if newSg != "" {
		instancePutIntoServerGroupRequest := &edgecloud.InstancePutIntoServerGroupRequest{ServerGroupID: newSg}
		task, _, err := client.Instances.PutIntoServerGroup(ctx, d.Id(), instancePutIntoServerGroupRequest)
		if err != nil {
			return fmt.Errorf("error when put the instance to new server group: %w", err)
		}

		if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
			return fmt.Errorf("error while waiting for instance put to new server group: %w", err)
		}
	}

	return nil
}

func changeVolumes(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	oldVolumesRaw, newVolumesRaw := d.GetChange("volume")
	oldVolumes, newVolumes := oldVolumesRaw.([]interface{}), newVolumesRaw.([]interface{})

	oldIDs := getVolumeIDsSet(oldVolumes)
	newIDs := getVolumeIDsSet(newVolumes)

	// detach volumes
	for volumeID := range converter.MapLeftDiff(oldIDs, newIDs) {
		volume := getVolumeInfoByID(volumeID, oldVolumes)
		if volume["boot_index"].(int) == 0 {
			return fmt.Errorf("cannot detach primary boot device with boot_index=0. id: %s", volumeID)
		}

		volumeDetachRequest := &edgecloud.VolumeDetachRequest{InstanceID: d.Id()}
		if _, _, err := client.Volumes.Detach(ctx, volumeID, volumeDetachRequest); err != nil {
			return fmt.Errorf("еrror while detaching the volume: %w", err)
		}
	}

	// attach volumes
	for volumeID := range converter.MapLeftDiff(newIDs, oldIDs) {
		volume := getVolumeInfoByID(volumeID, newVolumes)
		attachmentTag := volume["attachment_tag"].(string)

		switch volume["source"].(string) {
		case "image":
			return fmt.Errorf("cannot attach image-source volume, required 'existing-volume' or 'new-volume' source")
		case "existing-volume":
			volumeAttachRequest := &edgecloud.VolumeAttachRequest{
				InstanceID:    d.Id(),
				AttachmentTag: attachmentTag,
			}
			if _, _, err := client.Volumes.Attach(ctx, volume["volume_id"].(string), volumeAttachRequest); err != nil {
				return fmt.Errorf("cannot attach existing-volume: %s. error: %w", volumeID, err)
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
				return fmt.Errorf("error when creating a new instance volume: %w", err)
			}
			if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
				return fmt.Errorf("error while waiting for instance volume create: %w", err)
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
				return fmt.Errorf("error when extending instance volume: %w", err)
			}
			if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
				return fmt.Errorf("error while waiting for instance volume extend: %w", err)
			}
		}
	}

	return nil
}

func updateInterfaceSecurityGroups(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client, portID string, unAssignSGs, assignSGs []edgecloud.ID) error {
	for _, sg := range unAssignSGs {
		sgInfo, _, err := client.SecurityGroups.Get(ctx, sg.ID)
		if err != nil {
			return fmt.Errorf("cannot get security group with ID: %s, err: %w", sg.ID, err)
		}

		unAssignSecurityGroupRequest := &edgecloud.AssignSecurityGroupRequest{
			PortsSecurityGroupNames: []edgecloud.PortsSecurityGroupNames{
				{
					PortID:             portID,
					SecurityGroupNames: []string{sgInfo.Name},
				},
			},
		}
		if _, err = client.Instances.SecurityGroupUnAssign(ctx, d.Id(), unAssignSecurityGroupRequest); err != nil {
			return fmt.Errorf("cannot Unassign security group from Instance. SecGroup ID: %s, err: %w", sg.ID, err)
		}
	}

	for _, sg := range assignSGs {
		sgInfo, _, err := client.SecurityGroups.Get(ctx, sg.ID)
		if err != nil {
			return fmt.Errorf("cannot get security group with ID: %s, err: %w", sg.ID, err)
		}

		assignSecurityGroupRequest := &edgecloud.AssignSecurityGroupRequest{
			PortsSecurityGroupNames: []edgecloud.PortsSecurityGroupNames{
				{
					PortID:             portID,
					SecurityGroupNames: []string{sgInfo.Name},
				},
			},
		}
		if _, err := client.Instances.SecurityGroupAssign(ctx, d.Id(), assignSecurityGroupRequest); err != nil {
			return fmt.Errorf("cannot Assign security group to Instance. SecGroup ID: %s, err: %w", sg.ID, err)
		}
	}

	return nil
}

func detachInterface(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client, ifs map[string]interface{}) error {
	detachInterfaceRequest := &edgecloud.InstanceDetachInterfaceRequest{
		PortID:    ifs["port_id"].(string),
		IPAddress: ifs["ip_address"].(string),
	}
	task, _, err := client.Instances.DetachInterface(ctx, d.Id(), detachInterfaceRequest)
	if err != nil {
		return fmt.Errorf("error detaching the instance interface: %w", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return fmt.Errorf("error waiting for the instance interface detach: %w", err)
	}

	return nil
}

func attachInterface(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client, ifs map[string]interface{}) error {
	iType := edgecloud.InterfaceType(ifs["type"].(string))
	attachInterfaceRequest := &edgecloud.InstanceAttachInterfaceRequest{Type: iType}

	switch iType { //nolint: exhaustive
	case edgecloud.InterfaceTypeSubnet:
		attachInterfaceRequest.SubnetID = ifs["subnet_id"].(string)
	case edgecloud.InterfaceTypeAnySubnet:
		attachInterfaceRequest.NetworkID = ifs["network_id"].(string)
	case edgecloud.InterfaceTypeReservedFixedIP:
		attachInterfaceRequest.PortID = ifs["port_id"].(string)
	}
	attachInterfaceRequest.SecurityGroups = getSecurityGroupsIDs(ifs["security_groups"].([]interface{}))

	task, _, err := client.Instances.AttachInterface(ctx, d.Id(), attachInterfaceRequest)
	if err != nil {
		return fmt.Errorf("error attaching the instance interface: %w", err)
	}

	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return fmt.Errorf("error waiting for the instance interface atach: %w", err)
	}

	return nil
}
