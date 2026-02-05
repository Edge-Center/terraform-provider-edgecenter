package edgecenter

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func flattenTaints(taints []edgecloudV2.MKaaSTaint) []map[string]interface{} { //nolint:unused
	result := make([]map[string]interface{}, 0, len(taints))
	for _, t := range taints {
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
