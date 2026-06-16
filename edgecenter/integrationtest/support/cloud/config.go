package cloud

func WithProjectRegion(projectID, regionID int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID,
		"region_id":  regionID,
	}
}

func WithMetadata(metadata map[string]string) map[string]interface{} {
	rawMetadata := make(map[string]interface{}, len(metadata))
	for key, value := range metadata {
		rawMetadata[key] = value
	}

	return map[string]interface{}{
		"metadata_map": rawMetadata,
	}
}

func WithProjectID(projectID int) map[string]interface{} {
	return map[string]interface{}{
		"project_id": projectID,
	}
}

func WithName(name string) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
	}
}

func WithSize(size int) map[string]interface{} {
	return map[string]interface{}{
		"size": size,
	}
}

func WithTypeName(typeName string) map[string]interface{} {
	return map[string]interface{}{
		"type_name": typeName,
	}
}

func Merge(base map[string]interface{}, parts ...map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range base {
		out[key] = value
	}

	for _, part := range parts {
		for key, value := range part {
			out[key] = value
		}
	}

	return out
}
