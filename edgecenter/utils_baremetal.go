package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func validateInterfaceBaremetalOpts(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) diag.Diagnostics {
	_, ifaceOptsRaw := d.GetChange(InterfaceField)
	ifaceOptsList := ifaceOptsRaw.([]interface{})

	err := checkIfaceBaremetalAttrCombinations(ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	err = checkSingleIsParentIfaceBaremetal(ifaceOptsList)
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

	err = checkParentIfaceEqualOrdertBaremetal(ifaceOptsList)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func checkIfaceBaremetalAttrCombinations(ifaces []interface{}) error {
	for _, ifs := range ifaces {
		ifsMap := ifs.(map[string]interface{})
		ifsType := ifsMap[TypeField].(string)
		subnetID := ifsMap[SubnetIDField].(string)
		networkID := ifsMap[NetworkIDField].(string)
		portID := ifsMap[PortIDField].(string)
		existingFipIDField := ifsMap[ExistingFipIDField].(string)
		fipSourceField := ifsMap[FipSourceField].(string)
		switch ifsType {
		case string(edgecloudV2.InterfaceTypeExternal):
			if subnetID != "" || networkID != "" || portID != "" || fipSourceField != "" || existingFipIDField != "" {
				return fmt.Errorf("you can't use \"%s\", \"%s\", \"%s\", \"%s\", \"%s\" attributes for \"%s\" interface type", NetworkIDField, SubnetIDField, ExistingFipIDField, FipSourceField, PortIDField, ifsType)
			}
		case string(edgecloudV2.InterfaceTypeSubnet):
			if subnetID == "" || networkID == "" {
				return fmt.Errorf("attributes \"%s\", \"%s\" must be set for \"%s\" interface type", NetworkIDField, SubnetIDField, ifsType)
			}

			err := checkFloatingIPBaremetalAttrCombinations(fipSourceField, existingFipIDField)
			if err != nil {
				return err
			}

		case string(edgecloudV2.InterfaceTypeAnySubnet):
			if subnetID != "" || portID != "" {
				return fmt.Errorf("you can't use \"%s\", \"%s\" attributes for \"%s\" interface type", SubnetIDField, PortIDField, ifsType)
			}

			err := checkFloatingIPBaremetalAttrCombinations(fipSourceField, existingFipIDField)
			if err != nil {
				return err
			}
		case string(edgecloudV2.InterfaceTypeReservedFixedIP):
			if portID == "" {
				return fmt.Errorf("attribute \"%s\" must be set for \"%s\" interface type", PortIDField, ifsType)
			}
			if subnetID != "" || networkID != "" {
				return fmt.Errorf("you can't use \"%s\", \"%s\" attributes for \"%s\" interface type", NetworkIDField, SubnetIDField, ifsType)
			}

			err := checkFloatingIPBaremetalAttrCombinations(fipSourceField, existingFipIDField)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkFloatingIPBaremetalAttrCombinations(fipSourceField string, existingFipIDField string) error {
	switch fipSourceField {
	case string(edgecloudV2.NewFloatingIP):
		if existingFipIDField != "" {
			return fmt.Errorf("you can't use \"%s\" attribute for \"%s\" floating IP", ExistingFipIDField, edgecloudV2.NewFloatingIP)
		}
	case string(edgecloudV2.ExistingFloatingIP):
		if existingFipIDField == "" {
			return fmt.Errorf("attributes \"%s\" must be set for \"%s\" floating IP", ExistingFipIDField, edgecloudV2.ExistingFloatingIP)
		}
	}

	return nil
}

func checkSingleIsParentIfaceBaremetal(interfaces []interface{}) error {
	var isParentIfsCount int
	for _, ifs := range interfaces {
		ifsMap := ifs.(map[string]interface{})
		isParentRaw := ifsMap[IsParentField]
		isParent := isParentRaw.(bool)
		if isParent {
			isParentIfsCount++
		}
	}

	if len(interfaces) != 0 && isParentIfsCount != 1 {
		return fmt.Errorf("you must always have exactly one interface with set attribute 'is_parent = true'")
	}

	return nil
}

func checkParentIfaceEqualOrdertBaremetal(interfaces []interface{}) error {
	for _, ifs := range interfaces {
		ifsMap := ifs.(map[string]interface{})
		isParentRaw := ifsMap[IsParentField]
		orderRaw := ifsMap[OrderField]
		if isParentRaw.(bool) && orderRaw.(int) != 0 {
			return errors.New("parent interface must always have an order of 0(zero)")
		}
	}

	return nil
}

func convertApiIfaceToTfIface(apiIFaces []edgecloudV2.InstancePortInterface) ([]interface{}, error) {
	iFaces := make([]interface{}, 0)
	iFaceOrder := 0

	if len(apiIFaces) != 1 {
		return nil, fmt.Errorf("only one trunk interfces is allowed to baremetal instance")
	}

	iFace := apiIFaces[0]

	parentIFace := make(map[string]interface{})
	if len(iFace.IPAssignments) == 0 {
		return nil, fmt.Errorf("no IP assignments found in trunk interface")
	}

	parentIFace[IsParentField] = true
	parentIFace[OrderField] = iFaceOrder
	parentIFace[IPAddressField] = iFace.IPAssignments[0].IPAddress.String()

	if len(iFace.FloatingIPDetails) != 0 {
		// As we cannot retrieve the setting that we specified during creation from CloudAPI, we have set the default to 'existing'.
		parentIFace[FipSourceField] = string(edgecloudV2.ExistingFloatingIP)
		parentIFace[ExistingFipIDField] = iFace.FloatingIPDetails[0].ID
	}

	switch {
	case iFace.NetworkDetails.External:
		parentIFace[TypeField] = string(edgecloudV2.InterfaceTypeExternal)
	case strings.Contains(iFace.Name, "reserved_fixed_ip"):
		parentIFace[TypeField] = string(edgecloudV2.InterfaceTypeReservedFixedIP)
		parentIFace[PortIDField] = iFace.PortID
	default:
		parentIFace[TypeField] = string(edgecloudV2.InterfaceTypeSubnet)
		parentIFace[SubnetIDField] = iFace.IPAssignments[0].SubnetID
		parentIFace[NetworkIDField] = iFace.NetworkID
	}

	iFaces = append(iFaces, parentIFace)

	for _, iSub := range iFace.SubPorts {
		if len(iSub.IPAssignments) == 0 {
			continue
		}

		iFaceOrder++

		iSubFace := make(map[string]interface{})

		iSubFace[IsParentField] = false
		iSubFace[OrderField] = iFaceOrder

		iSubFace[IPAddressField] = iSub.IPAssignments[0].IPAddress.String()

		if len(iSub.FloatingIPDetails) != 0 {
			// As we cannot retrieve the setting that we specified during creation from CloudAPI, we have set the default to 'existing'.
			iSubFace[FipSourceField] = string(edgecloudV2.ExistingFloatingIP)
			iSubFace[ExistingFipIDField] = iSub.FloatingIPDetails[0].ID
		}

		switch {
		case iSub.NetworkDetails.External:
			iSubFace[TypeField] = string(edgecloudV2.InterfaceTypeExternal)
		case strings.Contains(iFace.Name, "reserved_fixed_ip"):
			iSubFace[TypeField] = string(edgecloudV2.InterfaceTypeReservedFixedIP)
			iSubFace[PortIDField] = iSub.PortID
		default:
			iSubFace[TypeField] = string(edgecloudV2.InterfaceTypeSubnet)
			iSubFace[SubnetIDField] = iSub.IPAssignments[0].SubnetID
			iSubFace[NetworkIDField] = iSub.NetworkID
		}

		iFaces = append(iFaces, iSubFace)
	}

	return iFaces, nil
}

// SafeGet returns a slice element by index and a boolean flag indicating the success of the operation.
func SafeGet[T any](slice []T, index int) (T, bool) {
	var zero T
	if index >= 0 && index < len(slice) {
		return slice[index], true
	}
	return zero, false // Возвращаем значение по умолчанию и false, если индекс вне диапазона
}
