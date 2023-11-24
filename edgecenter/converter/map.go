package converter

import "fmt"

// MapInterfaceToMapString converts a map[string]interface{} to map[string]string.
func MapInterfaceToMapString(m map[string]interface{}) map[string]string {
	mapString := make(map[string]string)

	for key, value := range m {
		mapString[key] = fmt.Sprintf("%v", value)
	}

	return mapString
}
