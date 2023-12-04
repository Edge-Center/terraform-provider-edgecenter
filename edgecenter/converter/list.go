package converter

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

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
		var I edgecloud.InstanceInterface
		if err := MapStructureDecoder(&I, &inter, decoderConfig); err != nil {
			return nil, err
		}

		floatingIPList := inter["floating_ip"].(*schema.Set).List()
		if len(floatingIPList) > 0 {
			fip := floatingIPList[0].(map[string]interface{})
			if fip["source"].(string) == "new" {
				I.FloatingIP.Source = edgecloud.NewFloatingIP
			} else {
				I.FloatingIP = &edgecloud.InterfaceFloatingIP{
					Source:             edgecloud.ExistingFloatingIP,
					ExistingFloatingID: fip["existing_floating_id"].(string),
				}
			}
		}

		sgList := inter["security_groups"].([]interface{})
		if len(sgList) > 0 {
			sgs := make([]edgecloud.ID, 0, len(sgList))
			for _, sg := range sgList {
				sgs = append(sgs, edgecloud.ID{ID: sg.(string)})
			}
			I.SecurityGroups = sgs
		}

		ifs[idx] = I
	}

	return ifs, nil
}
