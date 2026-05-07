package edgecenter

import (
	"context"
	"fmt"

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
	result := make([]interface{}, 0, len(taints))
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

func ValidateUniqueTaintKeys(set *schema.Set) error {
	seen := make(map[string]struct{}, set.Len())
	for _, item := range set.List() {
		m := item.(map[string]interface{})
		key := m["key"].(string)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate taint key %q: taint keys must be unique within a pool", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func customMKaaSPoolDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if raw, ok := d.GetOk(MKaaSPoolTaintsField); ok {
		if err := ValidateUniqueTaintKeys(raw.(*schema.Set)); err != nil {
			return err
		}
	}

	minPtr, maxPtr, autoscale := expandScalePolicy(d)
	if autoscale {
		if *maxPtr < *minPtr {
			return fmt.Errorf("scale_policy.auto_scale.max (%d) must be >= scale_policy.auto_scale.min (%d)",
				*maxPtr, *minPtr)
		}
	}

	return nil
}

// scalePolicyReader is the minimal surface for reading the nested
// scale_policy block. *schema.ResourceData and *schema.ResourceDiff both
// satisfy it.
type scalePolicyReader interface {
	GetOk(string) (interface{}, bool)
}

// expandScalePolicy returns (min, max, enabled). enabled is true if a
// scale_policy { auto_scale { ... } } block is present in config.
func expandScalePolicy(d scalePolicyReader) (*int, *int, bool) {
	raw, ok := d.GetOk(MKaaSPoolScalePolicyField)
	if !ok {
		return nil, nil, false
	}
	spList, ok := raw.([]interface{})
	if !ok || len(spList) == 0 || spList[0] == nil {
		return nil, nil, false
	}
	spMap, ok := spList[0].(map[string]interface{})
	if !ok {
		return nil, nil, false
	}
	asList, ok := spMap[MKaaSPoolAutoScaleField].([]interface{})
	if !ok || len(asList) == 0 || asList[0] == nil {
		return nil, nil, false
	}
	asMap, ok := asList[0].(map[string]interface{})
	if !ok {
		return nil, nil, false
	}
	minVal := asMap[MKaaSPoolMinField].(int)
	maxVal := asMap[MKaaSPoolMaxField].(int)
	return &minVal, &maxVal, true
}

// setPoolNodeCount writes node_count into state, leaving it empty when the autoscaler owns the value.
func setPoolNodeCount(d *schema.ResourceData, pool *edgecloudV2.MKaaSPool) {
	if pool.AutoscalingEnabled {
		_ = d.Set(MKaaSNodeCountField, nil)
		return
	}
	_ = d.Set(MKaaSNodeCountField, pool.NodeCount)
}

// flattenScalePolicy emits the nested scale_policy shape only when pool.AutoscalingEnabled is true.
func flattenScalePolicy(pool *edgecloudV2.MKaaSPool) []interface{} {
	if pool == nil || !pool.AutoscalingEnabled {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			MKaaSPoolAutoScaleField: []interface{}{
				map[string]interface{}{
					MKaaSPoolMinField:              pool.MinNodeCount,
					MKaaSPoolMaxField:              pool.MaxNodeCount,
					MKaaSPoolCurrentNodeCountField: pool.NodeCount,
				},
			},
		},
	}
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
