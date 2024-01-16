package edgecenter

import (
	"fmt"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func PrepareMetadata(apiMetadataRaw interface{}) (map[string]string, []map[string]interface{}) {
	metadataMap := make(map[string]string)
	var metadataReadOnly []map[string]interface{}
	// ToDo Delete after migrate to Go Client V2
	switch apiMetadata := apiMetadataRaw.(type) {
	case []metadata.Metadata:
		metadataReadOnly = make([]map[string]interface{}, 0, len(apiMetadata))
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
	case []edgecloudV2.MetadataDetailed:
		metadataReadOnly = make([]map[string]interface{}, 0, len(apiMetadata))
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

func MapInterfaceToMapString(mapInterface interface{}) (*edgecloudV2.Metadata, error) {
	mapString := make(edgecloudV2.Metadata)

	switch v := mapInterface.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", mapInterface)
	case map[string]interface{}:
		for key, value := range v {
			mapString[key] = fmt.Sprintf("%v", value)
		}
	case map[interface{}]interface{}:
		for key, value := range v {
			mapString[fmt.Sprintf("%v", key)] = fmt.Sprintf("%v", value)
		}
	}

	return &mapString, nil
}
