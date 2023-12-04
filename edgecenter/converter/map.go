package converter

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

var decoderConfig = &mapstructure.DecoderConfig{TagName: "json"}

// MapInterfaceToMapString converts a map[string]interface{} to map[string]string.
func MapInterfaceToMapString(m map[string]interface{}) map[string]string {
	mapString := make(map[string]string)

	for key, value := range m {
		mapString[key] = fmt.Sprintf("%v", value)
	}

	return mapString
}

// MapStructureDecoder decodes the given map into the provided structure using the specified decoder configuration.
func MapStructureDecoder(strct interface{}, v *map[string]interface{}, config *mapstructure.DecoderConfig) error {
	config.Result = strct
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	return decoder.Decode(*v)
}

// MapLeftDiff returns all elements in Left that are not in Right.
func MapLeftDiff(left, right map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for l := range left {
		if _, ok := right[l]; !ok {
			out[l] = struct{}{}
		}
	}

	return out
}

// MapsIntersection returns all elements in Left that are in Right.
func MapsIntersection(left, right map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for l := range left {
		if _, ok := right[l]; ok {
			out[l] = struct{}{}
		}
	}

	return out
}
