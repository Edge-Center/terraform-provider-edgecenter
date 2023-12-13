package instance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func setVolumes(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	// desc sorting: first volume is the last added
	volumes, _, err := client.Volumes.List(ctx, &edgecloud.VolumeListOptions{InstanceID: d.Id()})
	if err != nil {
		return fmt.Errorf("error retrieving instance volumes: %w", err)
	}

	// asc sorting: first volume is the first added
	currentVolumes := d.Get("volume").([]interface{})
	for i, v := range currentVolumes {
		volume := v.(map[string]interface{})
		volumeID := getVolumeIDByName(volume["name"].(string), volumes)
		if volumeID == "" {
			return fmt.Errorf("error during get volume id")
		}
		currentVolumes[i].(map[string]interface{})["id"] = volumeID
	}

	return d.Set("volume", currentVolumes)
}

func setInterfaces(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	instancePorts, _, err := client.Instances.PortsList(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("error retrieving instance ports: %w", err)
	}

	routerInterfaces, _, err := client.Instances.InterfaceList(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("error retrieving instance interfaces: %w", err)
	}

	currentInterfaces := d.Get("interface").([]interface{})
	for i, v := range currentInterfaces {
		portID := routerInterfaces[i].PortID

		var sgList []string
		for _, port := range instancePorts {
			if port.ID == portID {
				for _, sg := range port.SecurityGroups {
					sgList = append(sgList, sg.ID)
				}
			}
		}

		ipAssignments := routerInterfaces[i].IPAssignments
		if len(ipAssignments) == 0 {
			continue
		}

		fip := routerInterfaces[i].FloatingIPDetails

		ifs := v.(map[string]interface{})
		ifsType := edgecloud.InterfaceType(ifs["type"].(string))
		switch ifsType { //nolint: exhaustive
		case edgecloud.InterfaceTypeSubnet, edgecloud.InterfaceTypeAnySubnet:
			currentInterfaces[i].(map[string]interface{})["subnet_id"] = ipAssignments[0].SubnetID
		}
		currentInterfaces[i].(map[string]interface{})["port_id"] = portID
		currentInterfaces[i].(map[string]interface{})["ip_address"] = ipAssignments[0].IPAddress.String()
		if len(fip) > 0 {
			currentInterfaces[i].(map[string]interface{})["floating_ip"] = fip[0].FloatingIPAddress
		}

		currentInterfaces[i].(map[string]interface{})["security_groups"] = sgList
	}

	return d.Set("interface", currentInterfaces)
}

func setAddresses(_ context.Context, d *schema.ResourceData, instance *edgecloud.Instance) error {
	addresses := make([]map[string]string, 0, len(instance.Addresses))
	for networkName, networkInfo := range instance.Addresses {
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

	return d.Set("addresses", addresses)
}

func setFlavor(_ context.Context, d *schema.ResourceData, instance *edgecloud.Instance) error {
	flavor := map[string]interface{}{
		"flavor_name": instance.Flavor.FlavorName,
		"vcpus":       strconv.Itoa(instance.Flavor.VCPUS),
		"ram":         strconv.Itoa(instance.Flavor.RAM),
		"flavor_id":   instance.Flavor.FlavorID,
	}

	return d.Set("flavor", flavor)
}

func setSecurityGroups(_ context.Context, d *schema.ResourceData, instance *edgecloud.Instance) error {
	if len(instance.SecurityGroups) > 0 {
		securityGroups := make([]string, 0, len(instance.SecurityGroups))
		for _, sg := range instance.SecurityGroups {
			securityGroups = append(securityGroups, sg.Name)
		}
		return d.Set("security_groups", securityGroups)
	}

	return nil
}

func setMetadataDetailed(_ context.Context, d *schema.ResourceData, instance *edgecloud.Instance) error {
	if len(instance.MetadataDetailed) > 0 {
		metadata := make([]map[string]interface{}, 0, len(instance.MetadataDetailed))
		for _, metadataItem := range instance.MetadataDetailed {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}

		return d.Set("metadata_detailed", metadata)
	}

	return nil
}
