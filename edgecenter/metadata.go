package edgecenter

import "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/meta"

func PrepareMetadata(apiMetadataRaw interface{}) (map[string]string, []map[string]interface{}) {
	return meta.PrepareMetadata(apiMetadataRaw)
}

func PrepareMetadataReadonly(apiMetadataRaw interface{}) []map[string]interface{} {
	return meta.PrepareMetadataReadonly(apiMetadataRaw)
}

func MapInterfaceToMapString(mapInterface interface{}) (*map[string]string, error) {
	return meta.MapInterfaceToMapString(mapInterface) //nolint:wrapcheck
}
