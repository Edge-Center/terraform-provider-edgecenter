package edgecenter

import (
	"context"
	"fmt"
	"slices"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type baremetalIfaceMap = map[string]interface{}

type sourceIfacesMap = map[string]sourceIface

type sourceIface struct {
	ipAddress     string
	isParent      bool
	networkID     string
	networkName   string
	portID        string
	subnetID      string
	isExternal    bool
	existingFipID string
}

func validateBaremetalInterfaceConfig(ctx context.Context, interfaces baremetalIfaceMap, clientV2 *edgecloudV2.Client) error {
	ifsType := interfaces["type"].(string)

	switch ifsType {
	case types.ExternalInterfaceType.String():
		if interfaces["subnet_id"].(string) != "" || interfaces["network_id"].(string) != "" || interfaces["port_id"].(string) != "" {
			return fmt.Errorf("prohibit the use of any port or network_id or subnet_id with interface type \"%s\"", ifsType)
		}

	case types.AnySubnetInterfaceType.String():
		if interfaces["subnet_id"].(string) != "" || interfaces["port_id"].(string) != "" {
			return fmt.Errorf("prohibit the use of any subnet or any port with interface type \"%s\"", ifsType)
		}

		network, _, err := clientV2.Networks.Get(ctx, interfaces["network_id"].(string))
		if err != nil {
			return fmt.Errorf("error getting network information: %s", err.Error())
		}

		if network.Type == "vxlan" {
			return fmt.Errorf("vxLAN networks are not supported for baremetal instances")
		}

	case types.ReservedFixedIPType.String():
		if interfaces["subnet_id"].(string) != "" || interfaces["network_id"].(string) != "" {
			return fmt.Errorf("prohibit the use of any subnet or network with interface type \"%s\"", ifsType)
		}

	case types.SubnetInterfaceType.String():
		if interfaces["port_id"].(string) != "" {
			return fmt.Errorf("prohibit the use of any port with interface type \"%s\"", ifsType)
		}

		if interfaces["network_id"].(string) != "" {
			networkID := interfaces["network_id"].(string)
			subnetID := interfaces["subnet_id"].(string)

			network, _, err := clientV2.Networks.Get(ctx, interfaces["network_id"].(string))
			if err != nil {
				return fmt.Errorf("error getting network information: %s", err.Error())
			}

			if network.Type == "vxlan" {
				return fmt.Errorf("vxLAN networks are not supported for baremetal instances")
			}

			if !slices.Contains(network.Subnets, subnetID) {
				return fmt.Errorf("subnet with ID: \"%s\" does not belong to the network with ID: \"%s\"", subnetID, networkID)
			}
		}
	}

	return nil
}

func prepareBaremetalInstanceInterfaceCreateOpts(ctx context.Context, clientV2 *edgecloudV2.Client, interfaces []interface{}) ([]edgecloudV2.BareMetalInterfaceOpts, error) {
	interfaceOptsList := make([]edgecloudV2.BareMetalInterfaceOpts, len(interfaces))
	for i, iFace := range interfaces {
		raw := iFace.(baremetalIfaceMap)

		if err := validateBaremetalInterfaceConfig(ctx, raw, clientV2); err != nil {
			return nil, fmt.Errorf("validate interface config error: %s", err.Error())
		}

		interfaceOpts := edgecloudV2.BareMetalInterfaceOpts{
			Type:      edgecloudV2.InterfaceType(raw["type"].(string)),
			NetworkID: raw["network_id"].(string),
			SubnetID:  raw["subnet_id"].(string),
			PortID:    raw["port_id"].(string),
		}

		fipSource := raw["fip_source"].(string)
		fipID := raw["existing_fip_id"].(string)

		if fipSource != "" {
			interfaceOpts.FloatingIP = &edgecloudV2.InterfaceFloatingIP{
				Source:             edgecloudV2.FloatingIPSource(fipSource),
				ExistingFloatingID: fipID,
			}
		}

		interfaceOptsList[i] = interfaceOpts
	}

	return interfaceOptsList, nil
}

func convertSourceBaremetalInterfaceToMap(iface edgecloudV2.InstancePortInterface) (sourceIfacesMap, error) {
	sim := make(sourceIfacesMap)

	if len(iface.IPAssignments) == 0 {
		return nil, fmt.Errorf("interface IPAssignments for network ID: %s not found", iface.NetworkID)
	}

	parentIface := sourceIface{
		isParent:    true,
		networkID:   iface.NetworkID,
		portID:      iface.PortID,
		isExternal:  iface.NetworkDetails.External,
		networkName: iface.NetworkDetails.Name,
		subnetID:    iface.IPAssignments[0].SubnetID,
	}

	if len(iface.FloatingIPDetails) != 0 {
		parentIface.ipAddress = iface.FloatingIPDetails[0].FloatingIPAddress
		parentIface.existingFipID = iface.FloatingIPDetails[0].ID
	} else {
		parentIface.ipAddress = iface.IPAssignments[0].IPAddress.String()
	}

	if parentIface.isExternal {
		sim["external"] = parentIface
	} else {
		sim[parentIface.networkID] = parentIface
		sim[parentIface.subnetID] = parentIface
		sim[parentIface.portID] = parentIface
	}

	for _, sp := range iface.SubPorts {
		if len(sp.IPAssignments) == 0 {
			return nil, fmt.Errorf("interface IPAssignments for network ID: %s not found", sp.NetworkID)
		}

		subIface := sourceIface{
			isParent:    false,
			networkID:   sp.NetworkID,
			portID:      sp.PortID,
			isExternal:  sp.NetworkDetails.External,
			networkName: sp.NetworkDetails.Name,
			subnetID:    sp.IPAssignments[0].SubnetID,
		}

		if len(sp.FloatingIPDetails) != 0 {
			subIface.ipAddress = sp.FloatingIPDetails[0].FloatingIPAddress
			subIface.existingFipID = sp.FloatingIPDetails[0].ID
		} else {
			subIface.ipAddress = sp.IPAssignments[0].IPAddress.String()
		}

		if subIface.isExternal {
			sim["external"] = subIface
		} else {
			sim[subIface.networkID] = subIface
			sim[subIface.subnetID] = subIface
			sim[subIface.portID] = subIface
		}
	}

	return sim, nil
}

func updateInterfaceState(iface baremetalIfaceMap, sim sourceIfacesMap) {
	typeIface := iface["type"].(string)
	fipSourceType := iface["fip_source"].(string)

	var data sourceIface
	switch typeIface {
	case types.SubnetInterfaceType.String():
		data = sim[iface["subnet_id"].(string)]
		iface["subnet_id"] = data.subnetID
		iface["network_id"] = data.networkID

	case types.AnySubnetInterfaceType.String():
		data = sim[iface["network_id"].(string)]
		iface["network_id"] = data.networkID

	case types.ReservedFixedIPType.String():
		data = sim[iface["port_id"].(string)]
		iface["port_id"] = data.portID

	case types.ExternalInterfaceType.String():
		data = sim["external"]
	}

	iface["ip_address"] = data.ipAddress
	iface["network_name"] = data.networkName
	iface["is_parent_readonly"] = data.isParent
	iface["port_id_readonly"] = data.portID

	if data.existingFipID != "" && fipSourceType == "existing" {
		iface["existing_fip_id"] = data.existingFipID
	}
}

// isInterfaceContains checks if a given verifiable interface is present in the provided set of interfaces (ifsSet).
func isInterfaceContains(verifiable map[string]interface{}, ifsSet []interface{}) bool {
	verifiableType := verifiable["type"].(string)
	verifiableSubnetID, _ := verifiable["subnet_id"].(string)
	verifiableNetworkID, _ := verifiable["network_id"].(string)

	for _, e := range ifsSet {
		i := e.(map[string]interface{})
		iType := i["type"].(string)

		subnetID, _ := i["subnet_id"].(string)
		networkID, _ := i["network_id"].(string)
		if iType == types.ExternalInterfaceType.String() && verifiableType == types.ExternalInterfaceType.String() {
			return true
		}

		if iType == types.SubnetInterfaceType.String() && verifiableType == types.SubnetInterfaceType.String() && subnetID == verifiableSubnetID {
			return true
		}

		if iType == types.AnySubnetInterfaceType.String() && verifiableType == types.AnySubnetInterfaceType.String() && networkID == verifiableNetworkID {
			return true
		}
	}

	return false
}
