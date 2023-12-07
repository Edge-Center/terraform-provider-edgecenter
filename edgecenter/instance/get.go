package instance

import edgecloud "github.com/Edge-Center/edgecentercloud-go"

func getVolumeIDsSet(volumes []interface{}) map[string]struct{} {
	ids := make(map[string]struct{}, len(volumes))
	for _, volumeRaw := range volumes {
		volume := volumeRaw.(map[string]interface{})
		ids[volume["id"].(string)] = struct{}{}
	}

	return ids
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

func getVolumeIDByName(name string, volumeList []edgecloud.Volume) string {
	for _, volume := range volumeList {
		if volume.Name == name {
			return volume.ID
		}
	}

	return ""
}

func getVolumesBootIndexList(volumes []interface{}) []int {
	idxList := make([]int, 0, len(volumes))
	for _, volumeRaw := range volumes {
		volume := volumeRaw.(map[string]interface{})
		idxList = append(idxList, volume["boot_index"].(int))
	}

	return idxList
}

// getSecurityGroupsIDs converts a slice of raw security group IDs to a slice of edgecloud.ItemID.
func getSecurityGroupsIDs(sgsRaw []interface{}) []edgecloud.ID {
	sgs := make([]edgecloud.ID, len(sgsRaw))
	for i, sgID := range sgsRaw {
		sgs[i] = edgecloud.ID{ID: sgID.(string)}
	}
	return sgs
}

// getSecurityGroupsDifference finds the difference between two slices of edgecloud.ID.
func getSecurityGroupsDifference(sl1, sl2 []edgecloud.ID) (diff []edgecloud.ID) { //nolint: nonamedreturns
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
