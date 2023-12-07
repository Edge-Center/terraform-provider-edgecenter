package converter

import (
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func ListInterfaceToListInstanceVolumeCreate(volumes []interface{}) ([]edgecloud.InstanceVolumeCreate, error) {
	vols := make([]edgecloud.InstanceVolumeCreate, len(volumes))
	for i, volume := range volumes {
		vol := volume.(map[string]interface{})
		var V edgecloud.InstanceVolumeCreate
		if err := MapStructureDecoder(&V, &vol, decoderConfig); err != nil {
			return nil, err
		}
		vols[i] = V
	}

	return vols, nil
}

func ListInterfaceToListInstanceInterface(interfaces []interface{}) ([]edgecloud.InstanceInterface, error) {
	ifs := make([]edgecloud.InstanceInterface, len(interfaces))
	for idx, i := range interfaces {
		inter := i.(map[string]interface{})
		I := edgecloud.InstanceInterface{
			Type:      edgecloud.InterfaceType(inter["type"].(string)),
			NetworkID: inter["network_id"].(string),
			PortID:    inter["port_id"].(string),
			SubnetID:  inter["subnet_id"].(string),
		}

		switch inter["floating_ip_source"].(string) {
		case "new":
			I.FloatingIP = &edgecloud.InterfaceFloatingIP{
				Source: edgecloud.NewFloatingIP,
			}
		case "existing":
			I.FloatingIP = &edgecloud.InterfaceFloatingIP{
				Source:             edgecloud.ExistingFloatingIP,
				ExistingFloatingID: inter["floating_ip"].(string),
			}
		default:
			I.FloatingIP = nil
		}

		sgList := inter["security_groups"].([]interface{})
		if len(sgList) > 0 {
			sgs := make([]edgecloud.ID, 0, len(sgList))
			for _, sg := range sgList {
				sgs = append(sgs, edgecloud.ID{ID: sg.(string)})
			}
			I.SecurityGroups = sgs
		} else {
			I.SecurityGroups = []edgecloud.ID{}
		}

		ifs[idx] = I
	}

	return ifs, nil
}
