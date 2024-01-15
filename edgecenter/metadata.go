package edgecenter

import (
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

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

func PrepareMetadataReadonly(apiMetadataRaw interface{}) []map[string]interface{} {
	// ToDo Delete after migrate to Go Client V2
	var metadataReadOnly []map[string]interface{}
	switch apiMetadata := apiMetadataRaw.(type) {
	case []metadata.Metadata:
		metadataReadOnly = make([]map[string]interface{}, 0, len(apiMetadata))
		if len(apiMetadata) > 0 {
			for _, metadataItem := range apiMetadata {
				metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
					"key":       metadataItem.Key,
					"value":     metadataItem.Value,
					"read_only": metadataItem.ReadOnly,
				})
			}
		}
	case []edgecloudV2.MetadataDetailed:
		metadataReadOnly = make([]map[string]interface{}, 0, len(apiMetadata))
		if len(apiMetadata) > 0 {
			for _, metadataItem := range apiMetadata {
				metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
					"key":       metadataItem.Key,
					"value":     metadataItem.Value,
					"read_only": metadataItem.ReadOnly,
				})
			}
		}
	}

	return metadataReadOnly
}
