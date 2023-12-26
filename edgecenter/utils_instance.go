package edgecenter

import (
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
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/port/v1/ports"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/servergroup/v1/servergroups"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
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

// extractInstanceInterfaceToListCreate creates a list of InterfaceInstanceCreateOpts objects from a list of interfaces.
func extractInstanceInterfaceToListCreate(interfaces []interface{}) ([]instances.InterfaceInstanceCreateOpts, error) {
	interfaceInstanceCreateOptsList := make([]instances.InterfaceInstanceCreateOpts, 0)
	for _, iFace := range interfaces {
		iFaceMap := iFace.(map[string]interface{})

		interfaceOpts, err := decodeInstanceInterfaceOpts(iFaceMap)
		if err != nil {
			return nil, err
		}

		rawSgsID := iFaceMap["security_groups"].([]interface{})
		sgs := make([]edgecloud.ItemID, len(rawSgsID))
		for i, sgID := range rawSgsID {
			sgs[i] = edgecloud.ItemID{ID: sgID.(string)}
		}

		interfaceInstanceCreateOpts := instances.InterfaceInstanceCreateOpts{
			InterfaceOpts:  interfaceOpts,
			SecurityGroups: sgs,
		}
		interfaceInstanceCreateOptsList = append(interfaceInstanceCreateOptsList, interfaceInstanceCreateOpts)
	}

	return interfaceInstanceCreateOptsList, nil
}

// extractInstanceInterfaceToListRead creates a list of InterfaceOpts objects from a list of interfaces.
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
func extractVolumesMap(volumes []interface{}) ([]instances.CreateVolumeOpts, error) {
	vols := make([]instances.CreateVolumeOpts, len(volumes))
	for i, volume := range volumes {
		vol := volume.(map[string]interface{})
		var V instances.CreateVolumeOpts
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

// ServerV2StateRefreshFunc returns a StateRefreshFunc to track the state of an instance using its instanceID.
func ServerV2StateRefreshFunc(client *edgecloud.ServiceClient, instanceID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		s, err := instances.Get(client, instanceID).Extract()
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

// findInstancePort searches for the instance port with the specified portID in the given list of instance ports.
func findInstancePort(portID string, ports []instances.InstancePorts) (instances.InstancePorts, error) {
	for _, port := range ports {
		if port.ID == portID {
			return port, nil
		}
	}

	return instances.InstancePorts{}, fmt.Errorf("port not found")
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

// detachInterfaceFromInstance detaches interface from an instance.
func detachInterfaceFromInstance(client *edgecloud.ServiceClient, instanceID string, iface map[string]interface{}) error {
	var opts instances.InterfaceOpts
	opts.PortID = iface["port_id"].(string)
	opts.IPAddress = iface["ip_address"].(string)

	log.Printf("[DEBUG] detach interface: %+v", opts)
	results, err := instances.DetachInterface(client, instanceID, opts).Extract()
	if err != nil {
		return err
	}

	err = tasks.WaitTaskAndProcessResult(client, results.Tasks[0], true, InstanceCreatingTimeout, func(task tasks.TaskID) error {
		if taskInfo, err := tasks.Get(client, string(task)).Extract(); err != nil {
			return fmt.Errorf("cannot get task with ID: %s. Error: %w, task: %+v", task, err, taskInfo)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// attachInterfaceToInstance attach interface to instance.
func attachInterfaceToInstance(instanceClient, portsClient *edgecloud.ServiceClient, instanceID string, iface map[string]interface{}) error {
	iType := types.InterfaceType(iface["type"].(string))
	opts := instances.InterfaceInstanceCreateOpts{
		InterfaceOpts: instances.InterfaceOpts{Type: iType},
	}

	switch iType { // nolint: exhaustive
	case types.SubnetInterfaceType:
		opts.SubnetID = iface["subnet_id"].(string)
	case types.AnySubnetInterfaceType:
		opts.NetworkID = iface["network_id"].(string)
	case types.ReservedFixedIPType:
		opts.PortID = iface["port_id"].(string)
	}
	opts.SecurityGroups = getSecurityGroupsIDs(iface["security_groups"].([]interface{}))

	log.Printf("[DEBUG] attach interface: %+v", opts)
	results, err := instances.AttachInterface(instanceClient, instanceID, opts).Extract()
	if err != nil {
		return fmt.Errorf("cannot attach interface: %s. Error: %w", iType, err)
	}

	err = tasks.WaitTaskAndProcessResult(instanceClient, results.Tasks[0], true, InstanceCreatingTimeout, func(task tasks.TaskID) error {
		taskInfo, err := tasks.Get(instanceClient, string(task)).Extract()
		if err != nil {
			return fmt.Errorf("cannot get task with ID: %s. Error: %w, task: %+v", task, err, taskInfo)
		}

		if _, err := instances.ExtractInstancePortIDFromTask(taskInfo); err != nil {
			reservedFixedIPID, ok := (*taskInfo.Data)["reserved_fixed_ip_id"]
			if !ok || reservedFixedIPID.(string) == "" {
				return fmt.Errorf("cannot retrieve instance port ID from task info: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	interfacesListAPI, err := instances.ListInterfacesAll(instanceClient, instanceID)
	if err != nil {
		return fmt.Errorf("error from getting instance interfaces: %w", err)
	}

	if err = adjustPortSecurityDisabledOpt(portsClient, interfacesListAPI, iface); err != nil {
		return fmt.Errorf("cannot adjust port_security_disabled opt: %+v. Error: %w", iface, err)
	}

	return nil
}

// adjustPortSecurityDisabledOpt aligns the state of the interface (port_security_disabled) with what is specified in the
// iface["port_security_disabled"].
func adjustPortSecurityDisabledOpt(portsClient *edgecloud.ServiceClient, interfacesListAPI []instances.Interface, iface map[string]interface{}) error {
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
						if _, err := ports.DisablePortSecurity(portsClient, requestedIfacePortID).Extract(); err != nil {
							return err
						}
					case false:
						if _, err := ports.EnablePortSecurity(portsClient, requestedIfacePortID).Extract(); err != nil {
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

// deleteServerGroup removes a server group from an instance.
func deleteServerGroup(sgClient, instanceClient *edgecloud.ServiceClient, instanceID, sgID string) error {
	log.Printf("[DEBUG] remove server group from instance: %s", instanceID)
	results, err := instances.RemoveServerGroup(instanceClient, instanceID).Extract()
	if err != nil {
		return fmt.Errorf("failed to remove server group %s from instance %s: %w", sgID, instanceID, err)
	}

	err = tasks.WaitTaskAndProcessResult(sgClient, results.Tasks[0], true, InstanceCreatingTimeout, func(task tasks.TaskID) error {
		sgInfo, err := servergroups.Get(sgClient, sgID).Extract()
		if err != nil {
			return fmt.Errorf("failed to get server group %s: %w", sgID, err)
		}
		for _, instanceInfo := range sgInfo.Instances {
			if instanceInfo.InstanceID == instanceID {
				return fmt.Errorf("server group %s was not removed from instance %s", sgID, instanceID)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// addServerGroup adds a server group to an instance.
func addServerGroup(sgClient, instanceClient *edgecloud.ServiceClient, instanceID, sgID string) error {
	log.Printf("[DEBUG] add server group to instance: %s", instanceID)
	results, err := instances.AddServerGroup(instanceClient, instanceID, instances.ServerGroupOpts{ServerGroupID: sgID}).Extract()
	if err != nil {
		return fmt.Errorf("failed to add server group %s to instance %s: %w", sgID, instanceID, err)
	}

	err = tasks.WaitTaskAndProcessResult(sgClient, results.Tasks[0], true, InstanceCreatingTimeout, func(task tasks.TaskID) error {
		sgInfo, err := servergroups.Get(sgClient, sgID).Extract()
		if err != nil {
			return fmt.Errorf("cannot get server group with ID: %s. Error: %w", sgID, err)
		}
		for _, instanceInfo := range sgInfo.Instances {
			if instanceInfo.InstanceID == instanceID {
				return nil
			}
		}
		return fmt.Errorf("the server group: %s was not added to the instance: %s. Error: %w", sgID, instanceID, err)
	})

	if err != nil {
		return err
	}

	return nil
}

// removeSecurityGroupFromInstance removes one or more security groups from a specific instance port.
func removeSecurityGroupFromInstance(sgClient, instanceClient *edgecloud.ServiceClient, instanceID, portID string, removeSGs []edgecloud.ItemID) error {
	for _, sg := range removeSGs {
		sgInfo, err := securitygroups.Get(sgClient, sg.ID).Extract()
		if err != nil {
			return err
		}

		portSGNames := instances.PortSecurityGroupNames{PortID: &portID, SecurityGroupNames: []string{sgInfo.Name}}
		sgOpts := instances.SecurityGroupOpts{PortsSecurityGroupNames: []instances.PortSecurityGroupNames{portSGNames}}

		log.Printf("[DEBUG] remove security group opts: %+v", sgOpts)
		if err := instances.UnAssignSecurityGroup(instanceClient, instanceID, sgOpts).Err; err != nil {
			return fmt.Errorf("cannot remove security group. Error: %w", err)
		}
	}

	return nil
}

// attachSecurityGroupToInstance attaches one or more security groups to a specific instance port.
func attachSecurityGroupToInstance(sgClient, instanceClient *edgecloud.ServiceClient, instanceID, portID string, addSGs []edgecloud.ItemID) error {
	for _, sg := range addSGs {
		sgInfo, err := securitygroups.Get(sgClient, sg.ID).Extract()
		if err != nil {
			return err
		}

		portSGNames := instances.PortSecurityGroupNames{PortID: &portID, SecurityGroupNames: []string{sgInfo.Name}}
		sgOpts := instances.SecurityGroupOpts{PortsSecurityGroupNames: []instances.PortSecurityGroupNames{portSGNames}}

		log.Printf("[DEBUG] attach security group opts: %+v", sgOpts)
		if err := instances.AssignSecurityGroup(instanceClient, instanceID, sgOpts).Err; err != nil {
			return fmt.Errorf("cannot attach security group. Error: %w", err)
		}
	}

	return nil
}

// prepareSecurityGroups prepares a list of unique security groups assigned to all instance ports.
func prepareSecurityGroups(ports []instances.InstancePorts) []interface{} {
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
func getSecurityGroupsIDs(sgsRaw []interface{}) []edgecloud.ItemID {
	sgs := make([]edgecloud.ItemID, len(sgsRaw))
	for i, sgID := range sgsRaw {
		sgs[i] = edgecloud.ItemID{ID: sgID.(string)}
	}
	return sgs
}

// getSecurityGroupsDifference finds the difference between two slices of edgecloud.ItemID.
func getSecurityGroupsDifference(sl1, sl2 []edgecloud.ItemID) (diff []edgecloud.ItemID) { // nolint: nonamedreturns
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
