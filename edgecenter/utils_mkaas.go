package edgecenter

import edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"

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
