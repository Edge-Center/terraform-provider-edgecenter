package edgecenter

import "github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"

func PrepareMetadata(apiMetadata []metadata.Metadata) (map[string]string, []map[string]interface{}) {
	metadataMap := make(map[string]string)
	metadataReadOnly := make([]map[string]interface{}, 0, len(apiMetadata))

	if len(apiMetadata) > 0 {
		for _, metadataItem := range apiMetadata {
			if !metadataItem.ReadOnly {
				metadataMap[metadataItem.Key] = metadataItem.Value
			}
			metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
	}

	return metadataMap, metadataReadOnly
}

func PrepareMetadataReadonly(apiMetadata []metadata.Metadata) []map[string]interface{} {
	metadataReadOnly := make([]map[string]interface{}, 0, len(apiMetadata))

	if len(apiMetadata) > 0 {
		for _, metadataItem := range apiMetadata {
			metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
	}

	return metadataReadOnly
}
