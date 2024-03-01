package edgecenter

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

type instanceInterfaces []interface{}

type InstanceInterfaceWithIPAddress struct {
	InstanceInterface edgecloudV2.InstanceInterface
	IPAddress         string
}

func (s instanceInterfaces) Len() int {
	return len(s)
}

func (s instanceInterfaces) Less(i, j int) bool {
	ifLeft := s[i].(map[string]interface{})
	ifRight := s[j].(map[string]interface{})

	// only bm instance has a parent interface, and it should be attached first
	isTrunkLeft, okLeft := ifLeft["is_parent"]
	isTrunkRight, okRight := ifRight["is_parent"]
	if okLeft && okRight {
		left, _ := isTrunkLeft.(bool)
		right, _ := isTrunkRight.(bool)
		switch {
		case left && !right:
			return true
		case right && !left:
			return false
		}
	}

	lOrder, _ := ifLeft["order"].(int)
	rOrder, _ := ifRight["order"].(int)
	if lOrder != rOrder {
		return lOrder < rOrder
	}
	lPortID, _ := ifLeft["port_id"].(string)
	rPortID, _ := ifRight["port_id"].(string)

	return lPortID < rPortID
}

func (s instanceInterfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type OrderedInterfaceOpts struct {
	InstanceInterfaceWithIPAddress
	Order int
}

// decodeInstanceInterfaceOptsV2 decodes the interface and returns InstanceInterface with FloatingIP.
func decodeInstanceInterfaceOptsV2(iFaceMap map[string]interface{}) edgecloudV2.InstanceInterface {
	iFace := edgecloudV2.InstanceInterface{
		Type:      edgecloudV2.InterfaceType(iFaceMap["type"].(string)),
		NetworkID: iFaceMap["network_id"].(string),
		PortID:    iFaceMap["port_id"].(string),
		SubnetID:  iFaceMap["subnet_id"].(string),
	}
	switch iFaceMap["fip_source"].(string) {
	case "new":
		iFace.FloatingIP = &edgecloudV2.InterfaceFloatingIP{
			Source: edgecloudV2.NewFloatingIP,
		}
	case "existing":
		iFace.FloatingIP = &edgecloudV2.InterfaceFloatingIP{
			Source:             edgecloudV2.ExistingFloatingIP,
			ExistingFloatingID: iFaceMap["existing_fip_id"].(string),
		}
	}

	rawSgsID := iFaceMap["security_groups"]
	if rawSgsID == nil {
		return iFace
	}
	rawSgsIDList := iFaceMap["security_groups"].([]interface{})
	sgs := make([]edgecloudV2.ID, len(rawSgsIDList))
	for i, sgID := range rawSgsIDList {
		sgs[i] = edgecloudV2.ID{ID: sgID.(string)}
	}
	iFace.SecurityGroups = sgs

	return iFace
}

// extractInstanceInterfaceToListCreateV2 creates a list of InstanceInterface objects from a list of interfaces.
func extractInstanceInterfaceToListCreateV2(interfaces []interface{}) []edgecloudV2.InstanceInterface {
	interfaceInstanceCreateOptsList := make([]edgecloudV2.InstanceInterface, 0)
	for _, tfIFace := range interfaces {
		iFaceMap := tfIFace.(map[string]interface{})
		iFace := decodeInstanceInterfaceOptsV2(iFaceMap)
		interfaceInstanceCreateOptsList = append(interfaceInstanceCreateOptsList, iFace)
	}

	return interfaceInstanceCreateOptsList
}

// extractInstanceInterfaceToListReadV2 creates a list of InterfaceOpts objects from a list of interfaces.
func extractInstanceInterfaceToListReadV2(interfaces []interface{}) map[string]OrderedInterfaceOpts {
	orderedInterfacesMap := make(map[string]OrderedInterfaceOpts)
	for _, iFace := range interfaces {
		var instanceInterfaceWithIPAddress InstanceInterfaceWithIPAddress
		if iFace == nil {
			continue
		}

		iFaceMap := iFace.(map[string]interface{})
		interfaceOpts := decodeInstanceInterfaceOptsV2(iFaceMap)
		instanceInterfaceWithIPAddress.InstanceInterface = interfaceOpts
		instanceInterfaceWithIPAddress.IPAddress = iFaceMap["ip_address"].(string)
		order, _ := iFaceMap["order"].(int)
		orderedInt := OrderedInterfaceOpts{instanceInterfaceWithIPAddress, order}
		orderedInterfacesMap[instanceInterfaceWithIPAddress.InstanceInterface.SubnetID] = orderedInt
		orderedInterfacesMap[instanceInterfaceWithIPAddress.InstanceInterface.NetworkID] = orderedInt
		orderedInterfacesMap[instanceInterfaceWithIPAddress.InstanceInterface.PortID] = orderedInt
		if instanceInterfaceWithIPAddress.InstanceInterface.Type == edgecloudV2.InterfaceTypeExternal {
			orderedInterfacesMap[string(instanceInterfaceWithIPAddress.InstanceInterface.Type)] = orderedInt
		}
	}

	return orderedInterfacesMap
}

// extractKeyValueV2 takes a slice of metadata interfaces and converts it into an edgecloudV2.Metadata structure.
func extractKeyValueV2(metadata []interface{}) (map[string]interface{}, error) {
	metaData := make(map[string]interface{}, len(metadata))
	for _, meta := range metadata {
		md, err := MapInterfaceToMapString(meta)
		if err != nil {
			return nil, err
		}
		mdVal := *md
		metaData[mdVal["key"]] = mdVal["value"]
	}
	return metaData, nil
}

// volumeUniqueID generates a unique ID for a volume based on its volume_id attribute.
func volumeUniqueID(i interface{}) int {
	e := i.(map[string]interface{})
	h := md5.New()
	io.WriteString(h, e["volume_id"].(string))
	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

// isInterfaceAttachedV2 checks if an interface is attached to a list of instances.Interface objects based on the subnet ID or external interface type.
func isInterfaceAttachedV2(ifs []edgecloudV2.InstancePortInterface, ifs2 map[string]interface{}) bool {
	subnetID, _ := ifs2["subnet_id"].(string)
	iType := types.InterfaceType(ifs2["type"].(string))
	for _, i := range ifs {
		if iType == types.ExternalInterfaceType && i.NetworkDetails.External {
			return true
		}
		for _, assignment := range i.IPAssignments {
			if assignment.SubnetID == subnetID {
				return true
			}
		}
		for _, subPort := range i.SubPorts {
			if iType == types.ExternalInterfaceType && subPort.NetworkDetails.External {
				return true
			}
			for _, assignment := range subPort.IPAssignments {
				if assignment.SubnetID == subnetID {
					return true
				}
			}
		}
	}

	return false
}

// isInterfaceContains checks if a given verifiable interface is present in the provided set of interfaces (ifsSet).
func isInterfaceContains(verifiable map[string]interface{}, ifsSet []interface{}) bool {
	verifiableType := verifiable["type"].(string)
	verifiableSubnetID, _ := verifiable["subnet_id"].(string)
	for _, e := range ifsSet {
		i := e.(map[string]interface{})
		iType := i["type"].(string)
		subnetID, _ := i["subnet_id"].(string)
		if iType == types.ExternalInterfaceType.String() && verifiableType == types.ExternalInterfaceType.String() {
			return true
		}

		if iType == verifiableType && subnetID == verifiableSubnetID {
			return true
		}
	}

	return false
}

// ServerV2StateRefreshFuncV2 returns a StateRefreshFunc to track the state of an instance using its instanceID.
func ServerV2StateRefreshFuncV2(ctx context.Context, client *edgecloudV2.Client, instanceID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		s, _, err := client.Instances.Get(ctx, instanceID)
		if err != nil {
			var errDefault404 edgecloud.Default404Error
			if errors.As(err, &errDefault404) {
				return s, "DELETED", nil
			}
			return nil, "", err
		}

		return s, s.VMState, nil
	}
}

// findInstancePortV2 searches for the instance port with the specified portID in the given list of instance ports.
func findInstancePortV2(portID string, ports []edgecloudV2.InstancePort) (edgecloudV2.InstancePort, error) {
	for _, port := range ports {
		if port.ID == portID {
			return port, nil
		}
	}

	return edgecloudV2.InstancePort{}, fmt.Errorf("port not found")
}

// contains check if slice contains the element.
func contains[K comparable](slice []K, elm K) bool {
	for _, s := range slice {
		if s == elm {
			return true
		}
	}
	return false
}

// getMapDifference compares two maps and returns a map of only different values.
// uncheckedKeys - list of keys to skip when comparing.
func getMapDifference(iMapOld, iMapNew map[string]interface{}, uncheckedKeys []string) map[string]interface{} {
	differentFields := make(map[string]interface{})

	for oldMapK, oldMapV := range iMapOld {
		if contains(uncheckedKeys, oldMapK) {
			continue
		}

		if newMapV, ok := iMapNew[oldMapK]; !ok || !reflect.DeepEqual(newMapV, oldMapV) {
			differentFields[oldMapK] = oldMapV
		}
	}

	for newMapK, newMapV := range iMapNew {
		if contains(uncheckedKeys, newMapK) {
			continue
		}

		if _, ok := iMapOld[newMapK]; !ok {
			differentFields[newMapK] = newMapV
		}
	}

	return differentFields
}

// detachInterfaceFromInstanceV2 detaches interface from an instance.
func detachInterfaceFromInstanceV2(ctx context.Context, client *edgecloudV2.Client, instanceID string, iface map[string]interface{}) error {
	var opts edgecloudV2.InstanceDetachInterfaceRequest
	opts.PortID = iface["port_id"].(string)
	opts.IPAddress = iface["ip_address"].(string)

	log.Printf("[DEBUG] detach interface: %+v", opts)

	result, _, err := client.Instances.DetachInterface(ctx, instanceID, &opts)
	if err != nil {
		return fmt.Errorf("error from detaching interface. instsnceID %s, opts: %v, err: %w", instanceID, opts, err)
	}

	taskID := result.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID)
	if err != nil {
		return err
	}

	if task.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot detach intreface  with opts: %v", opts)
	}

	return nil
}

// attachInterfaceToInstance attach interface to instance.
func attachInterfaceToInstanceV2(ctx context.Context, client *edgecloudV2.Client, instanceID string, iface map[string]interface{}) error {
	iType := edgecloudV2.InterfaceType(iface["type"].(string))
	opts := edgecloudV2.InstanceAttachInterfaceRequest{Type: iType}

	switch iType { // nolint: exhaustive
	case edgecloudV2.InterfaceTypeSubnet:
		opts.SubnetID = iface["subnet_id"].(string)
	case edgecloudV2.InterfaceTypeAnySubnet:
		opts.NetworkID = iface["network_id"].(string)
	case edgecloudV2.InterfaceTypeReservedFixedIP:
		opts.PortID = iface["port_id"].(string)
	}
	secGroups := iface["security_groups"]
	if secGroups != nil {
		opts.SecurityGroups = getSecurityGroupsIDsV2(secGroups.([]interface{}))
	} else {
		opts.SecurityGroups = []edgecloudV2.ID{}
	}
	log.Printf("[DEBUG] attach interface: %+v", opts)
	results, _, err := client.Instances.AttachInterface(ctx, instanceID, &opts)
	if err != nil {
		return fmt.Errorf("cannot attach interface: %s. Error: %w", iType, err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID)
	if err != nil {
		return err
	}

	if task.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot attach interface with opts: %v", opts)
	}

	_, isBareMetal := iface["is_parent"]

	if !isBareMetal {
		interfacesListAPI, _, err := client.Instances.InterfaceList(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("error from getting instance interfaces: %w", err)
		}

		if err = adjustPortSecurityDisabledOptV2(ctx, client, interfacesListAPI, iface); err != nil {
			return fmt.Errorf("cannot adjust port_security_disabled opt: %+v. Error: %w", iface, err)
		}
	}

	return nil
}

// adjustPortSecurityDisabledOptV2 aligns the state of the interface (port_security_disabled) with what is specified in the
// iface["port_security_disabled"].
func adjustPortSecurityDisabledOptV2(ctx context.Context, client *edgecloudV2.Client, interfacesListAPI []edgecloudV2.InstancePortInterface, iface map[string]interface{}) error {
	portSecurityDisabled := iface["port_security_disabled"].(bool)
	IPAddress := iface["ip_address"].(string)
	subnetID := iface["subnet_id"].(string)
	ifacePortID := iface["port_id"].(string)

LOOP:
	for _, iFace := range interfacesListAPI {
		if len(iFace.IPAssignments) == 0 {
			continue
		}

		requestedIfacePortID := iFace.PortID
		for _, assignment := range iFace.IPAssignments {
			requestedIfaceSubnetID := assignment.SubnetID
			requestedIfaceIPAddress := assignment.IPAddress.String()

			if subnetID == requestedIfaceSubnetID || IPAddress == requestedIfaceIPAddress || ifacePortID == requestedIfacePortID {
				if !iFace.PortSecurityEnabled != portSecurityDisabled {
					switch portSecurityDisabled {
					case true:
						if _, _, err := client.Ports.DisablePortSecurity(ctx, requestedIfacePortID); err != nil {
							return err
						}
					case false:
						if _, _, err := client.Ports.EnablePortSecurity(ctx, requestedIfacePortID); err != nil {
							return err
						}
					}
				}

				break LOOP
			}
		}
	}

	return nil
}

// deleteServerGroupV2 removes a server group from an instance.
func deleteServerGroupV2(ctx context.Context, client *edgecloudV2.Client, instanceID string) error {
	log.Printf("[DEBUG] remove server group from instance: %s", instanceID)

	results, _, err := client.Instances.RemoveFromServerGroup(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("error from removing server group. instanceId: %s, err: %w", instanceID, err)
	}
	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID)
	if err != nil {
		return err
	}

	if task.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot delete instance with ID %s from servergoup", instanceID)
	}

	return nil
}

// addServerGroupV2 adds a server group to an instance.
func addServerGroupV2(ctx context.Context, client *edgecloudV2.Client, instanceID, sgID string) error {
	log.Printf("[DEBUG] add server group to instance: %s", instanceID)

	results, _, err := client.Instances.PutIntoServerGroup(ctx, instanceID, &edgecloudV2.InstancePutIntoServerGroupRequest{ServerGroupID: sgID})
	if err != nil {
		return fmt.Errorf("failed to add server group %s to instance %s: %w", sgID, instanceID, err)
	}
	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID)
	if err != nil {
		return err
	}

	if task.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot add instance with id %s to servergroup %s", instanceID, sgID)
	}

	return nil
}

// removeSecurityGroupFromInstanceV2 removes one or more security groups from a specific instance port.
func removeSecurityGroupFromInstanceV2(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, removeSGs []edgecloudV2.ID) error {
	for _, sg := range removeSGs {
		sgInfo, _, err := client.SecurityGroups.Get(ctx, sg.ID)
		if err != nil {
			return err
		}

		portSGNames := edgecloudV2.PortsSecurityGroupNames{
			SecurityGroupNames: []string{sgInfo.Name},
			PortID:             portID,
		}
		sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

		log.Printf("[DEBUG] remove security group opts: %+v", sgOpts)
		if _, err := client.Instances.SecurityGroupUnAssign(ctx, instanceID, &sgOpts); err != nil {
			return fmt.Errorf("cannot remove security group. Error: %w", err)
		}
	}

	return nil
}

// attachSecurityGroupToInstance attaches one or more security groups to a specific instance port.
func attachSecurityGroupToInstanceV2(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, addSGs []edgecloudV2.ID) error {
	for _, sg := range addSGs {
		sgInfo, _, err := client.SecurityGroups.Get(ctx, sg.ID)
		if err != nil {
			return err
		}

		portSGNames := edgecloudV2.PortsSecurityGroupNames{
			SecurityGroupNames: []string{sgInfo.Name},
			PortID:             portID,
		}
		sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

		log.Printf("[DEBUG] attach security group opts: %+v", sgOpts)

		if _, err := client.Instances.SecurityGroupAssign(ctx, instanceID, &sgOpts); err != nil {
			return fmt.Errorf("cannot attach security group. Error: %w", err)
		}
	}

	return nil
}

// prepareSecurityGroupsV2 prepares a list of unique security groups assigned to all instance ports.
func prepareSecurityGroupsV2(ports []edgecloudV2.InstancePort) []interface{} {
	securityGroups := make(map[string]bool)
	for _, port := range ports {
		for _, sg := range port.SecurityGroups {
			securityGroups[sg.ID] = true
		}
	}

	result := make([]interface{}, 0, len(securityGroups))
	for sgID := range securityGroups {
		result = append(result, map[string]interface{}{
			"id":   sgID,
			"name": "",
		})
	}

	return result
}

// getSecurityGroupsIDs converts a slice of raw security group IDs to a slice of edgecloud.ItemID.
func getSecurityGroupsIDsV2(sgsRaw []interface{}) []edgecloudV2.ID {
	sgs := make([]edgecloudV2.ID, len(sgsRaw))
	for i, sgID := range sgsRaw {
		sgs[i] = edgecloudV2.ID{ID: sgID.(string)}
	}
	return sgs
}

// getSecurityGroupsDifferenceV2 finds the difference between two slices of edgecloudV2.ID.
func getSecurityGroupsDifferenceV2(sl1, sl2 []edgecloudV2.ID) (diff []edgecloudV2.ID) { // nolint: nonamedreturns
	set := make(map[string]bool)
	for _, item := range sl1 {
		set[item.ID] = true
	}

	for _, item := range sl2 {
		if !set[item.ID] {
			diff = append(diff, item)
		}
	}

	return diff
}

// changeVolumes execute code for function resourceInstanceUpdate.
func changeVolumes(ctx context.Context, d *schema.ResourceData, clientV2 *edgecloudV2.Client) error {
	oldVolumesRaw, newVolumesRaw := d.GetChange("volume")
	oldVolumes, newVolumes := oldVolumesRaw.([]interface{}), newVolumesRaw.([]interface{})

	oldIDs := getVolumeIDsSet(oldVolumes)
	newIDs := getVolumeIDsSet(newVolumes)

	// detach volumes
	for volumeID := range mapLeftDiff(oldIDs, newIDs) {
		volume := getVolumeInfoByID(volumeID, oldVolumes)
		if volume["boot_index"].(int) == 0 {
			return fmt.Errorf("cannot detach primary boot device with boot_index=0. id: %s", volumeID)
		}

		volumeDetachRequest := &edgecloudV2.VolumeDetachRequest{InstanceID: d.Id()}
		if _, _, err := clientV2.Volumes.Detach(ctx, volumeID, volumeDetachRequest); err != nil {
			return fmt.Errorf("еrror while detaching the volume: %w", err)
		}
	}

	// attach volumes
	for volumeID := range mapLeftDiff(newIDs, oldIDs) {
		volume := getVolumeInfoByID(volumeID, newVolumes)
		attachmentTag := volume["attachment_tag"].(string)

		switch volume["source"].(string) {
		case "existing-volume":
			volumeAttachRequest := &edgecloudV2.VolumeAttachRequest{
				InstanceID:    d.Id(),
				AttachmentTag: attachmentTag,
			}
			if _, _, err := clientV2.Volumes.Attach(ctx, volumeID, volumeAttachRequest); err != nil {
				return fmt.Errorf("cannot attach existing-volume: %s. error: %w", volumeID, err)
			}
		default:
			return fmt.Errorf("you cannot use such type: %s", volume["source"].(string))
		}
	}

	// resize the same volume
	for volumeID := range mapsIntersection(newIDs, oldIDs) {
		volumeOld := getVolumeInfoByID(volumeID, oldVolumes)
		volumeNew := getVolumeInfoByID(volumeID, newVolumes)

		if volumeOld["size"].(int) != volumeNew["size"].(int) {
			volumeExtendSizeRequest := &edgecloudV2.VolumeExtendSizeRequest{Size: volumeNew["size"].(int)}
			task, _, err := clientV2.Volumes.Extend(ctx, volumeID, volumeExtendSizeRequest)
			if err != nil {
				return fmt.Errorf("error when extending instance volume: %w", err)
			}
			if err = utilV2.WaitForTaskComplete(ctx, clientV2, task.Tasks[0]); err != nil {
				return fmt.Errorf("error while waiting for instance volume extend: %w", err)
			}
		}
	}

	return nil
}

// getVolumeIDsSet returns set of ids.
func getVolumeIDsSet(volumes []interface{}) map[string]struct{} {
	ids := make(map[string]struct{}, len(volumes))
	for _, volumeRaw := range volumes {
		volume := volumeRaw.(map[string]interface{})
		ids[volume["id"].(string)] = struct{}{}
	}

	return ids
}

// mapLeftDiff returns all elements in Left that are not in Right.
func mapLeftDiff(left, right map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for l := range left {
		if _, ok := right[l]; !ok {
			out[l] = struct{}{}
		}
	}

	return out
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

// mapsIntersection returns all elements in Left that are in Right.
func mapsIntersection(left, right map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for l := range left {
		if _, ok := right[l]; ok {
			out[l] = struct{}{}
		}
	}

	return out
}
