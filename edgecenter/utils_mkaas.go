package edgecenter

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func expandTaints(set *schema.Set) []edgecloudV2.MKaaSTaint {
	taints := make([]edgecloudV2.MKaaSTaint, 0, set.Len())
	for _, item := range set.List() {
		m := item.(map[string]interface{})
		taints = append(taints, edgecloudV2.MKaaSTaint{
			Key:    m["key"].(string),
			Value:  m["value"].(string),
			Effect: m["effect"].(string),
		})
	}

	return taints
}

func flattenTaints(taints []edgecloudV2.MKaaSTaint) []interface{} {
	sorted := make([]edgecloudV2.MKaaSTaint, len(taints))
	copy(sorted, taints)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Key != sorted[j].Key {
			return sorted[i].Key < sorted[j].Key
		}
		if sorted[i].Value != sorted[j].Value {
			return sorted[i].Value < sorted[j].Value
		}

		return sorted[i].Effect < sorted[j].Effect
	})
	result := make([]interface{}, 0, len(sorted))
	for _, t := range sorted {
		result = append(result, map[string]interface{}{
			"key":    t.Key,
			"value":  t.Value,
			"effect": t.Effect,
		})
	}

	return result
}

func mkaasClusterUnsupportedUpdateChanges(d *schema.ResourceData) []string {
	var unsupported []string

	topLevel := []string{
		MKaaSClusterKeypairNameField,
		MKaaSClusterPublishKubeAPIToInternet,
		NetworkIDField,
		SubnetIDField,
		ProjectIDField,
		ProjectNameField,
		RegionIDField,
		RegionNameField,
	}

	for _, f := range topLevel {
		if d.HasChange(f) {
			unsupported = append(unsupported, f)
		}
	}

	cpUnsupported := []string{
		FlavorField,
		MKaaSVolumeSizeField,
		MKaaSVolumeTypeField,
		MKaaSClusterVersionField,
	}

	for _, sf := range cpUnsupported {
		p := fmt.Sprintf("%s.%d.%s", MKaaSClusterControlPlaneField, 0, sf)
		if d.HasChange(p) {
			unsupported = append(unsupported, p)
		}
	}

	if d.HasChange(MKaaSClusterPodSubnetField) {
		unsupported = append(unsupported, MKaaSClusterPodSubnetField)
	}
	if d.HasChange(MKaaSClusterServiceSubnetField) {
		unsupported = append(unsupported, MKaaSClusterServiceSubnetField)
	}

	return unsupported
}

func mkaasPoolUnsupportedUpdateChanges(d *schema.ResourceData) []string {
	var unsupported []string

	fields := []string{
		ProjectIDField,
		ProjectNameField,
		RegionIDField,
		RegionNameField,
		MKaaSClusterIDField,
		FlavorField,
		MKaaSVolumeSizeField,
		MKaaSVolumeTypeField,
	}

	for _, f := range fields {
		if d.HasChange(f) {
			unsupported = append(unsupported, f)
		}
	}

	return unsupported
}
