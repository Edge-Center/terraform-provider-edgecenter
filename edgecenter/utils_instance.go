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
	"github.com/mitchellh/mapstructure"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

var instanceDecoderConfig = &mapstructure.DecoderConfig{
	TagName: "json",
}

type instanceInterfaces []interface{}

type InstancePortSecurityOpts struct {
	PortID               string
	PortSecurityDisabled bool
	SubnetID             string
	IPAddress            string
}

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

	return lOrder < rOrder
}

func (s instanceInterfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type OrderedInterfaceOpts struct {
	instances.InterfaceOpts
	Order int
}

// decodeInstanceInterfaceOpts decodes the interface and returns InterfaceOpts with FloatingIP.
func decodeInstanceInterfaceOpts(iFaceMap map[string]interface{}) (instances.InterfaceOpts, error) {
	var interfaceOpts instances.InterfaceOpts
	err := MapStructureDecoder(&interfaceOpts, &iFaceMap, instanceDecoderConfig)
	if err != nil {
		return interfaceOpts, err
	}

	if fipSource := iFaceMap["fip_source"].(string); fipSource != "" {
		var fip instances.CreateNewInterfaceFloatingIPOpts
		if existingFipID := iFaceMap["existing_fip_id"].(string); existingFipID != "" {
			fip.Source = types.ExistingFloatingIP
			fip.ExistingFloatingID = existingFipID
		} else {
			fip.Source = types.NewFloatingIP
		}
		interfaceOpts.FloatingIP = &fip
	}

	return interfaceOpts, nil
}

// decodeInstanceInterfaceOptsV2 decodes the interface and returns InstanceInterface with FloatingIP.
func decodeInstanceInterfaceOptsV2(iFaceMap map[string]interface{}) (edgecloudV2.InstanceInterface, error) {
	var interfaceOpts edgecloudV2.InstanceInterface
	err := MapStructureDecoder(&interfaceOpts, &iFaceMap, instanceDecoderConfig)
	if err != nil {
		return interfaceOpts, err
	}

	if fipSource := iFaceMap["fip_source"].(string); fipSource != "" {
		var fip edgecloudV2.InterfaceFloatingIP
		if existingFipID := iFaceMap["existing_fip_id"].(string); existingFipID != "" {
			fip.Source = edgecloudV2.ExistingFloatingIP
			fip.ExistingFloatingID = existingFipID
		} else {
			fip.Source = edgecloudV2.NewFloatingIP
		}
		interfaceOpts.FloatingIP = &fip
	}

	return interfaceOpts, nil
}

// extractInstanceInterfaceToListCreateV2 creates a list of InstanceInterface objects from a list of interfaces.
func extractInstanceInterfaceToListCreateV2(interfaces []interface{}) ([]edgecloudV2.InstanceInterface, error) {
	interfaceInstanceCreateOptsList := make([]edgecloudV2.InstanceInterface, 0)
	for _, iFace := range interfaces {
		iFaceMap := iFace.(map[string]interface{})

		interfaceOpts, err := decodeInstanceInterfaceOptsV2(iFaceMap)
		if err != nil {
			return nil, err
		}

		rawSgsID := iFaceMap["security_groups"].([]interface{})
		sgs := make([]edgecloudV2.ID, len(rawSgsID))
		for i, sgID := range rawSgsID {
			sgs[i] = edgecloudV2.ID{ID: sgID.(string)}
		}
		interfaceOpts.SecurityGroups = sgs
		interfaceInstanceCreateOptsList = append(interfaceInstanceCreateOptsList, interfaceOpts)
	}

	return interfaceInstanceCreateOptsList, nil
}

// extractInstanceInterfaceToListRead creates a list of InterfaceOpts objects from a list of interfaces.
// todo delete after migrate to v2 edgcentercloud-go client.
func extractInstanceInterfaceToListRead(interfaces []interface{}) ([]instances.InterfaceOpts, error) {
	interfaceOptsList := make([]instances.InterfaceOpts, 0)
	for _, iFace := range interfaces {
		if iFace == nil {
			continue
		}

		iFaceMap := iFace.(map[string]interface{})
		interfaceOpts, err := decodeInstanceInterfaceOpts(iFaceMap)
		if err != nil {
			return nil, err
		}
		interfaceOptsList = append(interfaceOptsList, interfaceOpts)
	}

	return interfaceOptsList, nil
}

// extractInstanceInterfaceToListReadV2 creates a list of InterfaceOpts objects from a list of interfaces.
func extractInstanceInterfaceToListReadV2(interfaces []interface{}) ([]InstanceInterfaceWithIPAddress, error) {
	interfaceOptsList := make([]InstanceInterfaceWithIPAddress, 0)
	for _, iFace := range interfaces {
		var instanceInterfaceWithIPAddress InstanceInterfaceWithIPAddress
		if iFace == nil {
			continue
		}

		iFaceMap := iFace.(map[string]interface{})
		interfaceOpts, err := decodeInstanceInterfaceOptsV2(iFaceMap)
		instanceInterfaceWithIPAddress.InstanceInterface = interfaceOpts
		instanceInterfaceWithIPAddress.IPAddress = iFaceMap["ip_address"].(string)

		if err != nil {
			return nil, err
		}
		interfaceOptsList = append(interfaceOptsList, instanceInterfaceWithIPAddress)
	}

	return interfaceOptsList, nil
}

// extractMetadataMap converts a map of metadata into a metadata set options structure.
func extractMetadataMap(metadata map[string]interface{}) instances.MetadataSetOpts {
	result := make([]instances.MetadataOpts, 0, len(metadata))
	for k, v := range metadata {
		result = append(result, instances.MetadataOpts{Key: k, Value: v.(string)})
	}
	return instances.MetadataSetOpts{Metadata: result}
}

// extractInstanceVolumesMap converts a slice of instance volumes into a map of volume IDs to boolean values.
func extractInstanceVolumesMap(volumes []interface{}) map[string]bool {
	result := make(map[string]bool)
	for _, volume := range volumes {
		v := volume.(map[string]interface{})
		result[v["volume_id"].(string)] = true
	}
	return result
}

// extractVolumesIntoMap converts a slice of volumes into a map with volume_id as the key.
func extractVolumesIntoMap(volumes []interface{}) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{}, len(volumes))
	for _, volume := range volumes {
		vol := volume.(map[string]interface{})
		result[vol["volume_id"].(string)] = vol
	}
	return result
}

// extractKeyValue takes a slice of metadata interfaces and converts it into an instances.MetadataSetOpts structure.
// todo delete after finish migrating to v2 edgecentercloud-go client.
func extractKeyValue(metadata []interface{}) (instances.MetadataSetOpts, error) {
	metaData := make([]instances.MetadataOpts, len(metadata))
	var metadataSetOpts instances.MetadataSetOpts
	for i, meta := range metadata {
		md := meta.(map[string]interface{})
		var MD instances.MetadataOpts
		err := MapStructureDecoder(&MD, &md, instanceDecoderConfig)
		if err != nil {
			return metadataSetOpts, err
		}
		metaData[i] = MD
	}
	metadataSetOpts.Metadata = metaData

	return metadataSetOpts, nil
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

// isInterfaceAttached checks if an interface is attached to a list of instances.Interface objects based on the subnet ID or external interface type.
func isInterfaceAttached(ifs []instances.Interface, ifs2 map[string]interface{}) bool {
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

// extractVolumesMap takes a slice of volume interfaces and converts it into a slice of instances.CreateVolumeOpts.
func extractVolumesMapV2(volumes []interface{}) ([]edgecloudV2.InstanceVolumeCreate, error) {
	vols := make([]edgecloudV2.InstanceVolumeCreate, len(volumes))
	for i, volume := range volumes {
		vol := volume.(map[string]interface{})
		var V edgecloudV2.InstanceVolumeCreate
		err := MapStructureDecoder(&V, &vol, instanceDecoderConfig)
		if err != nil {
			return nil, err
		}
		vols[i] = V
	}

	return vols, nil
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
	opts.SecurityGroups = getSecurityGroupsIDsV2(iface["security_groups"].([]interface{}))

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

	interfacesListAPI, _, err := client.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("error from getting instance interfaces: %w", err)
	}

	if err = adjustPortSecurityDisabledOptV2(ctx, client, interfacesListAPI, iface); err != nil {
		return fmt.Errorf("cannot adjust port_security_disabled opt: %+v. Error: %w", iface, err)
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
