package edgecenter

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/sync/errgroup"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

var instanceDecoderConfig = &mapstructure.DecoderConfig{
	TagName: "json",
}

type instanceInterfaces []interface{}

type instanceV2Interfaces []interface{}

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

type InstanceInterfaceMapWithIPAddress struct {
	InstanceInterface map[string]interface{}
	IPAddress         string
}

func (s instanceInterfaces) Len() int {
	return len(s)
}

func (s instanceV2Interfaces) Len() int {
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

func (s instanceV2Interfaces) Less(i, j int) bool { //nolint:revive
	ifRight := s[j].(map[string]interface{})

	isDefaultRight := ifRight[IsDefaultField].(bool)

	return !isDefaultRight
}

func (s instanceInterfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s instanceV2Interfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type OrderedInterfaceOpts struct {
	InstanceInterfaceWithIPAddress
	Order int
}

// decodeInstanceInterfaceOpts decodes the interface and returns InstanceInterface with FloatingIP.
func decodeInstanceInterfaceOpts(iFaceMap map[string]interface{}) edgecloudV2.InstanceInterface {
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

// decodeInstanceV2InterfaceOptsCreate decodes the interface and returns InstanceInterface with FloatingIP.
func decodeInstanceV2InterfaceOptsCreate(iFaceMap map[string]interface{}) edgecloudV2.InstanceInterface {
	iFace := edgecloudV2.InstanceInterface{
		Type:      edgecloudV2.InterfaceType(iFaceMap[TypeField].(string)),
		NetworkID: iFaceMap[NetworkIDField].(string),
		PortID:    iFaceMap[InstanceReservedFixedIPPortIDField].(string),
		SubnetID:  iFaceMap[SubnetIDField].(string),
	}
	sgs := make([]edgecloudV2.ID, 0, 1)
	iFace.SecurityGroups = sgs

	return iFace
}

// extractInstanceInterfaceToListCreate creates a list of InstanceInterface objects from a list of interfaces.
func extractInstanceInterfaceToListCreate(interfaces []interface{}) []edgecloudV2.InstanceInterface {
	interfaceInstanceCreateOptsList := make([]edgecloudV2.InstanceInterface, 0)
	for _, tfIFace := range interfaces {
		iFaceMap := tfIFace.(map[string]interface{})
		iFace := decodeInstanceInterfaceOpts(iFaceMap)
		interfaceInstanceCreateOptsList = append(interfaceInstanceCreateOptsList, iFace)
	}

	return interfaceInstanceCreateOptsList
}

// extractInstanceV2InterfaceOptsToListCreate creates a list of InstanceInterface objects from a list of interfaces.
func extractInstanceV2InterfaceOptsToListCreate(interfaces []interface{}) []edgecloudV2.InstanceInterface {
	interfaceInstanceCreateOptsList := make([]edgecloudV2.InstanceInterface, len(interfaces))
	for index, tfIFace := range interfaces {
		iFaceMap := tfIFace.(map[string]interface{})
		iFace := decodeInstanceV2InterfaceOptsCreate(iFaceMap)
		interfaceInstanceCreateOptsList[index] = iFace
	}
	return interfaceInstanceCreateOptsList
}

// prepareInstanceInterfaceCreateOpts prepares interface options for instance create request.
func prepareInstanceInterfaceCreateOpts(ctx context.Context, client *edgecloudV2.Client, interfaces []interface{}) ([]edgecloudV2.InstanceInterface, error) {
	ifsOpts := extractInstanceV2InterfaceOptsToListCreate(interfaces)
	defaultSG, err := utilV2.FindDefaultSG(ctx, client)
	if err != nil {
		return nil, err
	}
	for idx := range ifsOpts {
		ifsOpts[idx].SecurityGroups = []edgecloudV2.ID{{ID: defaultSG.ID}}
	}
	return ifsOpts, nil
}

// extractInstanceInterfaceToListRead creates a list of InterfaceOpts objects from a list of interfaces.
func extractInstanceInterfaceToListRead(interfaces []interface{}) map[string]OrderedInterfaceOpts {
	orderedInterfacesMap := make(map[string]OrderedInterfaceOpts)
	for _, iFace := range interfaces {
		var instanceInterfaceWithIPAddress InstanceInterfaceWithIPAddress
		if iFace == nil {
			continue
		}

		iFaceMap := iFace.(map[string]interface{})
		interfaceOpts := decodeInstanceInterfaceOpts(iFaceMap)
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

// extractInstanceV2InterfacesOptsToListRead creates a list of InterfaceOpts objects from a list of interfaces.
func extractInstanceV2InterfacesOptsToListRead(interfaces []interface{}) map[string]map[string]interface{} {
	interfacesMap := make(map[string]map[string]interface{})
	for _, iFace := range interfaces {
		if iFace == nil {
			continue
		}
		iFaceMap := iFace.(map[string]interface{})
		if networkID, ok := iFaceMap[NetworkIDField]; ok && networkID != "" {
			interfacesMap[networkID.(string)] = iFaceMap
		}

		if subnetID, ok := iFaceMap[SubnetIDField]; ok && subnetID != "" {
			interfacesMap[subnetID.(string)] = iFaceMap
		}

		if portID, ok := iFaceMap[PortIDField]; ok && portID != "" {
			interfacesMap[portID.(string)] = iFaceMap
		}
		if reservedFixedIPPortID, ok := iFaceMap[InstanceReservedFixedIPPortIDField]; ok && reservedFixedIPPortID != "" {
			interfacesMap[reservedFixedIPPortID.(string)] = iFaceMap
		}

		typeStr := iFaceMap[TypeField].(string)
		if edgecloudV2.InterfaceType(typeStr) == edgecloudV2.InterfaceTypeExternal {
			interfacesMap[typeStr] = iFaceMap
		}
	}

	return interfacesMap
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
	ans := int(binary.BigEndian.Uint32(h.Sum(nil)))
	return ans
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
		V.Source = edgecloudV2.VolumeSourceExistingVolume
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

// Temporary comment by CLOUDDEV-883
// findInstancePortV2 searches for the instance port with the specified portID in the given list of instance ports.
func findInstancePortV2(portID string, ports []edgecloudV2.InstancePort) (edgecloudV2.InstancePort, error) {
	for _, port := range ports {
		if port.ID == portID {
			return port, nil
		}
	}

	return edgecloudV2.InstancePort{}, fmt.Errorf("port not found")
}

// Temporary comment by CLOUDDEV-883
// contains check if slice contains the element.
func contains[K comparable](slice []K, elm K) bool {
	for _, s := range slice {
		if s == elm {
			return true
		}
	}
	return false
}

// Temporary comment by CLOUDDEV-883
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

	return nil
}

// attachInstanceV2InterfaceToInstance attach interface to instanceV2.
func attachInstanceV2InterfaceToInstance(ctx context.Context, client *edgecloudV2.Client, instanceID string, iface map[string]interface{}, defaultSG *edgecloudV2.SecurityGroup) error {
	iType := edgecloudV2.InterfaceType(iface[TypeField].(string))
	opts := edgecloudV2.InstanceAttachInterfaceRequest{
		Type:           iType,
		SecurityGroups: []edgecloudV2.ID{{ID: defaultSG.ID}},
	}

	switch iType { // nolint: exhaustive
	case edgecloudV2.InterfaceTypeSubnet:
		opts.SubnetID = iface[SubnetIDField].(string)
	case edgecloudV2.InterfaceTypeAnySubnet:
		opts.NetworkID = iface[NetworkIDField].(string)
	case edgecloudV2.InterfaceTypeReservedFixedIP:
		opts.PortID = iface[InstanceReservedFixedIPPortIDField].(string)
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

	return nil
}

// Temporary comment by CLOUDDEV-883
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

func adjustAllPortsSecurityDisabledOpt(ctx context.Context, client *edgecloudV2.Client, instanceID string, ifs []interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}
	interfacesListAPI, _, err := client.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error from getting instance interfaces: %w", err))
	}

	for _, iface := range ifs {
		ifaceMap := iface.(map[string]interface{})
		err = adjustPortSecurityDisabledOptV2(ctx, client, interfacesListAPI, ifaceMap)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error from port securtity disable option configuring. Interface: %#v, error: %w", ifaceMap, err))
		}
	}

	return diags
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

// Temporary comment by CLOUDDEV-883
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

// removeSecurityGroupsFromInstancePort removes one or more security groups from a specific instance port.
func removeSecurityGroupsFromInstancePort(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, removeSGIDs []interface{}) error {
	if len(removeSGIDs) == 0 {
		return nil
	}
	sgsToRemove := make([]string, 0, len(removeSGIDs))
	for _, sg := range removeSGIDs {
		sgsToRemove = append(sgsToRemove, sg.(string))
	}
	removeSGOpts, err := PrepareAndValidateAssignSecurityGroupRequestOpts(ctx, client, sgsToRemove, portID)
	if err != nil {
		return err
	}
	_, err = client.Instances.SecurityGroupUnAssign(ctx, instanceID, removeSGOpts)
	if err != nil {
		return err
	}

	return nil
}

// AssignSecurityGroupsToInstancePort assigns one or more security groups to a specific instance port.
func AssignSecurityGroupsToInstancePort(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, assignSGIDs []interface{}) error {
	if len(assignSGIDs) == 0 {
		return nil
	}
	sgsToAssign := make([]string, 0, len(assignSGIDs))
	for _, sg := range assignSGIDs {
		sgsToAssign = append(sgsToAssign, sg.(string))
	}

	removeSGOpts, err := PrepareAndValidateAssignSecurityGroupRequestOpts(ctx, client, sgsToAssign, portID)
	if err != nil {
		return err
	}
	_, err = client.Instances.SecurityGroupAssign(ctx, instanceID, removeSGOpts)
	if err != nil {
		return err
	}

	return nil
}

func PrepareAndValidateAssignSecurityGroupRequestOpts(ctx context.Context, client *edgecloudV2.Client, sgIDs []string, portID string) (*edgecloudV2.AssignSecurityGroupRequest, error) {
	filteredSGs, err := utilV2.SecurityGroupListByIDs(ctx, client, sgIDs)
	if err != nil {
		return nil, err
	}

	sgsNames := make([]string, len(filteredSGs))
	for idx, sg := range filteredSGs {
		sgsNames[idx] = sg.Name
	}

	portSGNames := edgecloudV2.PortsSecurityGroupNames{
		SecurityGroupNames: sgsNames,
		PortID:             portID,
	}

	sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

	return &sgOpts, nil
}

// Temporary comment by CLOUDDEV-883
// attachSecurityGroupToInstance attaches one or more security groups to a specific instance port.
func attachSecurityGroupToInstanceV2(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, addSGs []edgecloudV2.ID) error {
	if len(addSGs) == 0 {
		return nil
	}

	sgsNames := make([]string, 0, len(addSGs))
	for _, sg := range addSGs {
		sgInfo, _, err := client.SecurityGroups.Get(ctx, sg.ID)
		if err != nil {
			return err
		}
		sgsNames = append(sgsNames, sgInfo.Name)
	}

	sgOpts := edgecloudV2.AssignSecurityGroupRequest{
		PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{{
			SecurityGroupNames: sgsNames,
			PortID:             portID,
		}},
	}

	log.Printf("[DEBUG] attach security group opts: %+v", sgOpts)

	if _, err := client.Instances.SecurityGroupAssign(ctx, instanceID, &sgOpts); err != nil {
		return fmt.Errorf("cannot attach security group. Error: %w", err)
	}

	return nil
}

// Temporary comment by CLOUDDEV-883
// // prepareSecurityGroupsV2 prepares a list of unique security groups assigned to all instance ports.
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

// Temporary comment by CLOUDDEV-883
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

func validateInstanceResourceAttrs(d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}

	ifsDiags := validateInterfaceAttrs(d)
	if ifsDiags.HasError() {
		diags = append(diags, ifsDiags...)
	}

	return diags
}

func validateInstanceV2ResourceAttrs(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}

	ifsDiags := validateInterfaceOpts(ctx, client, d)
	if ifsDiags.HasError() {
		diags = append(diags, ifsDiags...)
	}

	bootVolumeDiags := validateBootVolumeAttrs(ctx, client, d)
	if bootVolumeDiags.HasError() {
		diags = append(diags, bootVolumeDiags...)
	}

	return diags
}

func validateInterfaceAttrs(d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	ifsRaw := d.Get("interface")
	ifsSlice := ifsRaw.([]interface{})
	for _, ifs := range ifsSlice {
		iNew := ifs.(map[string]interface{})
		var isPortSecDisabled, isSecGroupExists bool
		if v, ok := iNew["port_security_disabled"]; ok {
			isPortSecDisabled = v.(bool)
		}
		if v, ok := iNew["security_groups"]; ok {
			secGroups := v.([]interface{})
			if len(secGroups) != 0 {
				isSecGroupExists = true
			}
		}
		if isPortSecDisabled && isSecGroupExists {
			curDiag := diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("if attribute \"port_security_disabled\" for interface %+v set true, you can't set \"security_groups\" attribute", iNew),
				Detail:        "",
				AttributePath: nil,
			}
			diags = append(diags, curDiag)
		}
	}

	return diags
}

func validateInterfaceOpts(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) diag.Diagnostics {
	ifaceOptsRaw := d.Get(InstanceInterfacesField)
	ifaceOptsSet := ifaceOptsRaw.(*schema.Set)
	ifaceOptsList := ifaceOptsSet.List()

	err := checkIfaceAttrCombinations(ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	err = checkSingleDefaultIface(ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	err = checkSingleExternalIface(ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	err = checkUniqueIfaceSubnets(ctx, client, ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func validateBootVolumeAttrs(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	bootVolumes := d.Get("boot_volumes").(*schema.Set).List()
	bootVolumesMap := extractVolumesIntoMap(bootVolumes)

	err := CheckUniqueSequentialBootIndexes(bootVolumesMap)
	if err != nil {
		diags = diag.FromErr(err)
	}

	err = CheckAllImagesIsBootable(ctx, clientV2, bootVolumesMap)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

// VolumeV2StateRefreshFuncV2 returns a StateRefreshFunc to track the state of attaching volume using its volumeID.
func VolumeV2StateRefreshFuncV2(ctx context.Context, client *edgecloudV2.Client, volumeID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		volume, _, err := client.Volumes.Get(ctx, volumeID)
		if err != nil {
			var errDefault404 edgecloud.Default404Error
			if errors.As(err, &errDefault404) {
				return volume, "DELETED", nil
			}
			return nil, "", err
		}
		attachments := volume.Attachments
		if len(attachments) == 0 {
			return volume, "", errors.New("volume is not attached yet")
		}

		return volume, volume.Attachments[0].ServerID, nil
	}
}

func EnrichVolumeData(instanceVolumes []edgecloudV2.Volume, volumesState map[string]map[string]interface{}) []interface{} {
	enrichedVolumesData := make([]interface{}, 0, len(instanceVolumes))
	for _, vol := range instanceVolumes {
		v := make(map[string]interface{})
		stateVol, ok := volumesState[vol.ID]
		if !ok {
			continue
		}
		if bootIndex, bootIndexOk := stateVol["boot_index"]; bootIndexOk {
			v["boot_index"] = bootIndex
		}
		if attachTag, attachTagOk := stateVol["attachment_tag"]; attachTagOk {
			v["attachment_tag"] = attachTag
		}
		v["volume_id"] = vol.ID
		v["type_name"] = vol.VolumeType
		v["size"] = vol.Size
		v["name"] = vol.Name
		enrichedVolumesData = append(enrichedVolumesData, v)
	}

	return enrichedVolumesData
}

func PrepareVolumesDataToSet(instanceVolumes []edgecloudV2.Volume) (bootVolumes, dataVolumes []interface{}) { // nolint:nonamedreturns
	bootVolumesData := make([]interface{}, 0, len(instanceVolumes))
	dataVolumesData := make([]interface{}, 0, len(instanceVolumes))
	for _, vol := range instanceVolumes {
		v := make(map[string]interface{})
		v["volume_id"] = vol.ID
		v["type_name"] = vol.VolumeType
		v["size"] = vol.Size
		v["name"] = vol.Name
		switch vol.Bootable {
		case true:
			bootVolumesData = append(bootVolumesData, v)
		case false:
			dataVolumesData = append(dataVolumesData, v)
		}
	}

	return bootVolumesData, dataVolumesData
}

func UpdateVolumes(ctx context.Context, d *schema.ResourceData, client *edgecloudV2.Client, instanceID string, oldVolumesRaw, newVolumesRaw interface{}) error {
	oldVolumes := extractInstanceVolumesMap(oldVolumesRaw.(*schema.Set).List())
	newVolumes := extractInstanceVolumesMap(newVolumesRaw.(*schema.Set).List())

	vDetachOpts := edgecloudV2.VolumeDetachRequest{InstanceID: d.Id()}
	for vid := range oldVolumes {
		if isAttached := newVolumes[vid]; isAttached {
			// mark as already attached
			newVolumes[vid] = false
			continue
		}
		if _, _, err := client.Volumes.Detach(ctx, vid, &vDetachOpts); err != nil {
			return err
		}
	}

	// range over not attached volumes
	vAttachOpts := edgecloudV2.VolumeAttachRequest{InstanceID: d.Id()}
	for vid, ok := range newVolumes {
		if ok {
			if _, _, err := client.Volumes.Attach(ctx, vid, &vAttachOpts); err != nil {
				return err
			}
			startStateConf := &retry.StateChangeConf{
				Target:     []string{instanceID},
				Refresh:    VolumeV2StateRefreshFuncV2(ctx, client, vid),
				Timeout:    d.Timeout(schema.TimeoutUpdate),
				Delay:      2 * time.Second,
				MinTimeout: 3 * time.Second,
			}
			_, err := startStateConf.WaitForStateContext(ctx)
			if err != nil {
				return fmt.Errorf("error waiting for volume (%s) to become attached: %w", vid, err)
			}
		}
	}

	return nil
}

func CheckUniqueSequentialBootIndexes(volumes map[string]map[string]interface{}) error {
	viewedBootIndexes := make(map[int]string)
	sequentialBootIndexes := make(map[int]struct{})

	index := 0
	for range volumes {
		sequentialBootIndexes[index] = struct{}{}
		index++
	}

	for _, vol := range volumes {
		bootIndexRaw, ok := vol["boot_index"]
		if !ok {
			continue
		}
		bootIndex := bootIndexRaw.(int)
		if volID, ok := viewedBootIndexes[bootIndex]; ok {
			return fmt.Errorf("boot_index values must be unique, but boot_index %d for volume %s is duplicate", bootIndex, volID)
		}
		volumeID := vol["volume_id"].(string)
		if _, ok := sequentialBootIndexes[bootIndex]; !ok {
			return fmt.Errorf("boot_index values must be sequential, but boot_index %d for volume %s is not in available sequence", bootIndex, volumeID)
		}
		viewedBootIndexes[bootIndex] = volumeID
	}

	return nil
}

func checkSingleDefaultIface(interfaces []interface{}) error {
	var defaultIfsCount int
	for _, ifs := range interfaces {
		ifsMap := ifs.(map[string]interface{})
		isDefaultRaw := ifsMap[IsDefaultField]
		isDefault := isDefaultRaw.(bool)
		if isDefault {
			defaultIfsCount++
		}
	}
	if len(interfaces) != 0 && defaultIfsCount != 1 {
		return fmt.Errorf("you must always have exactly one interface with set attribute 'is_default = true'")
	}

	return nil
}

func checkSingleExternalIface(interfaces []interface{}) error {
	var externalIfsCount int
	for _, ifs := range interfaces {
		ifsMap := ifs.(map[string]interface{})
		ifaceType := ifsMap[TypeField].(string)
		if ifaceType == "external" {
			externalIfsCount++
		}
	}
	if externalIfsCount > 1 {
		return fmt.Errorf("you can have no more than one interface with the type 'external.'")
	}

	return nil
}

func checkUniqueIfaceSubnets(ctx context.Context, client *edgecloudV2.Client, interfaces []interface{}) error {
	subnets := make(map[string]interface{})

	reservedExists := slices.IndexFunc(interfaces, func(i interface{}) bool {
		ifsMap := i.(map[string]interface{})
		ifsType := ifsMap[TypeField].(string)
		return ifsType == string(edgecloudV2.InterfaceTypeReservedFixedIP)
	})

	var reservedIPS []edgecloudV2.ReservedFixedIP
	var err error
	if reservedExists >= 0 {
		reservedIPS, _, err = client.ReservedFixedIP.List(ctx, nil)
		if err != nil {
			return err
		}
	}

	for _, ifs := range interfaces {
		ifsMap := ifs.(map[string]interface{})
		ifaceType := ifsMap[TypeField].(string)
		switch ifaceType {
		case string(edgecloudV2.InterfaceTypeSubnet):
			subnetID := ifsMap[SubnetIDField].(string)
			if existingIface, ok := subnets[subnetID]; ok {
				return fmt.Errorf("%w:%#v and %#v", edgecloudV2.ErrMultipleIfaceWithSameSubnet, existingIface, ifs)
			}
			subnets[subnetID] = ifs
		case string(edgecloudV2.InterfaceTypeReservedFixedIP):
			portID := ifsMap[InstanceReservedFixedIPPortIDField].(string)
			reservedFixedIPIndex := slices.IndexFunc(reservedIPS, func(reservedFixedIP edgecloudV2.ReservedFixedIP) bool {
				return reservedFixedIP.PortID == portID
			})
			if reservedFixedIPIndex < 0 {
				return fmt.Errorf("%w: reserved_fixed_ip_port_id = %s", edgecloudV2.ErrResourceDoesntExist, portID)
			}
			reservedFixedIP := reservedIPS[reservedFixedIPIndex]
			if existingIface, ok := subnets[reservedFixedIP.SubnetID]; ok {
				return fmt.Errorf("%w:%#v and %#v", edgecloudV2.ErrMultipleIfaceWithSameSubnet, existingIface, ifs)
			}
			subnets[reservedFixedIP.SubnetID] = ifs
		}
	}

	return nil
}

func CheckAllImagesIsBootable(ctx context.Context, client *edgecloudV2.Client, volumes map[string]map[string]interface{}) error {
	group, groupCtx := errgroup.WithContext(ctx)
	for _, vol := range volumes {
		volumeID := vol["volume_id"].(string)
		group.Go(func() error {
			volume, _, err := client.Volumes.Get(groupCtx, volumeID)
			if err != nil {
				return err
			}
			if !volume.Bootable {
				return fmt.Errorf("volume %s in block `boot_volumes` is not bootable", volumeID)
			}
			return nil
		})
	}
	err := group.Wait()

	return err
}

func checkIfaceAttrCombinations(ifaces []interface{}) error {
	for _, ifs := range ifaces {
		ifsMap := ifs.(map[string]interface{})
		ifsType := ifsMap[TypeField].(string)
		subnetID := ifsMap[SubnetIDField].(string)
		networkID := ifsMap[NetworkIDField].(string)
		reservedFixedIPPortID := ifsMap[InstanceReservedFixedIPPortIDField].(string)
		switch ifsType {
		case string(edgecloudV2.InterfaceTypeReservedFixedIP):
			if reservedFixedIPPortID == "" {
				return fmt.Errorf("attribute \"%s\" must be set for \"%s\" interface type", InstanceReservedFixedIPPortIDField, ifsType)
			}
			if subnetID != "" || networkID != "" {
				return fmt.Errorf("you can't use \"%s\", \"%s\" attributes for \"%s\" interface type", NetworkIDField, SubnetIDField, ifsType)
			}
		case string(edgecloudV2.InterfaceTypeExternal):
			if subnetID != "" || networkID != "" || reservedFixedIPPortID != "" {
				return fmt.Errorf("you can't use \"%s\", \"%s\", \"%s\" attributes for \"%s\" interface type", NetworkIDField, SubnetIDField, InstanceReservedFixedIPPortIDField, ifsType)
			}
		case string(edgecloudV2.InterfaceTypeSubnet):
			if subnetID == "" || networkID == "" {
				return fmt.Errorf("attributes \"%s\", \"%s\" must be set for \"%s\" interface type", NetworkIDField, SubnetIDField, ifsType)
			}
			if reservedFixedIPPortID != "" {
				return fmt.Errorf("you can't use \"%s\" attribute for \"%s\" interface type", InstanceReservedFixedIPPortIDField, ifsType)
			}
		}
	}

	return nil
}

func prepareInterfacesOptsListFromAPI(interfacesListAPI []edgecloudV2.InstancePortInterface) []interface{} {
	var interfacesOptsList []interface{}

	for i, iFace := range interfacesListAPI {
		interfaceOptsMap := make(map[string]interface{})

		if len(iFace.IPAssignments) == 0 {
			continue
		}

		if i == 0 {
			interfaceOptsMap[IsDefaultField] = true
		}

		for _, assignment := range iFace.IPAssignments {
			switch {
			case iFace.NetworkDetails.External:
				interfaceOptsMap[TypeField] = string(edgecloudV2.InterfaceTypeExternal)

			case strings.Contains(iFace.Name, string(edgecloudV2.InterfaceTypeReservedFixedIP)):
				interfaceOptsMap[InstanceReservedFixedIPPortIDField] = iFace.PortID
				interfaceOptsMap[TypeField] = string(edgecloudV2.InterfaceTypeReservedFixedIP)

			default:
				interfaceOptsMap[TypeField] = string(edgecloudV2.InterfaceTypeSubnet)
				interfaceOptsMap[NetworkIDField] = iFace.NetworkID
				interfaceOptsMap[SubnetIDField] = assignment.SubnetID
			}

			interfaceOptsMap[PortIDField] = iFace.PortID
			interfaceOptsMap[IPAddressField] = assignment.IPAddress.String()
			interfaceOptsMap[NetworkNameField] = iFace.NetworkDetails.Name

			interfacesOptsList = append(interfacesOptsList, interfaceOptsMap)
		}
	}

	return interfacesOptsList
}

func prepareBootVolumesDataFromAPI(instanceVolumes []edgecloudV2.Volume) []interface{} {
	enrichedBootVolumesData := make([]interface{}, 0, len(instanceVolumes))

	i := 0
	for _, volume := range instanceVolumes {
		if !volume.Bootable {
			continue
		}

		newVolume := make(map[string]interface{})

		newVolume[NameField] = volume.Name
		newVolume[InstanceBootVolumesBootIndexField] = i
		newVolume[TypeNameField] = volume.VolumeType
		newVolume[InstanceVolumeSizeField] = volume.Size
		newVolume[InstanceVolumeIDField] = volume.ID

		enrichedBootVolumesData = append(enrichedBootVolumesData, newVolume)

		i++
	}

	return enrichedBootVolumesData
}

func prepareDataVolumesDataFromAPI(instanceVolumes []edgecloudV2.Volume) []interface{} {
	enrichedDataVolumesData := make([]interface{}, 0, len(instanceVolumes))

	for _, volume := range instanceVolumes {
		if volume.Bootable {
			continue
		}

		newVolume := make(map[string]interface{})

		newVolume[NameField] = volume.Name
		newVolume[TypeNameField] = volume.VolumeType
		newVolume[InstanceVolumeSizeField] = volume.Size
		newVolume[InstanceVolumeIDField] = volume.ID

		enrichedDataVolumesData = append(enrichedDataVolumesData, newVolume)
	}

	return enrichedDataVolumesData
}

func prepareMetadataFromAPI(ctx context.Context, clientV2 *edgecloudV2.Client, instanceID string) (map[string]interface{}, error) {
	newMetadata := make(map[string]interface{})

	mds, _, err := clientV2.Instances.MetadataList(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("cannot get metadata list. Error: %s", err.Error())
	}

	for _, md := range mds {
		if md.ReadOnly {
			continue
		}
		newMetadata[md.Key] = md.Value
	}

	return newMetadata, nil
}

// getInstance retrieves a Instance from the edge cloud service.
// It attempts to find the Instance either by its ID or by its name.
func getInstance(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Instance, error) {
	var (
		instance *edgecloudV2.Instance
		err      error
	)
	name := d.Get(NameField).(string)
	instanceID := d.Get(IDField).(string)

	switch {
	case instanceID != "":
		instance, _, err = clientV2.Instances.Get(ctx, instanceID)
		if err != nil {
			return nil, err
		}
	default:
		options := &edgecloudV2.InstanceListOptions{
			Name:             name,
			IncludeBaremetal: true,
		}

		insts, _, err := clientV2.Instances.List(ctx, options)
		if err != nil {
			return nil, err
		}

		foundInstances := make([]edgecloudV2.Instance, 0, len(insts))

		for _, i := range insts {
			if i.Name == name {
				foundInstances = append(foundInstances, i)
			}
		}

		switch {
		case len(foundInstances) == 0:
			return nil, errors.New("instance does not exist")

		case len(foundInstances) > 1:
			var message bytes.Buffer

			message.WriteString("Found instances:\n")

			for _, fI := range foundInstances {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", fI.ID))
			}

			return nil, fmt.Errorf("multiple instances found.\n %s.\n Use instance ID instead of name", message.String())
		}

		instance = &foundInstances[0]
	}

	return instance, nil
}
