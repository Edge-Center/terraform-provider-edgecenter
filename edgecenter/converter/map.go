package converter

import (
	"fmt"
	"reflect"

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

// contains check if slice contains the element.
func contains[K comparable](slice []K, elm K) bool {
	for _, s := range slice {
		if s == elm {
			return true
		}
	}

	return false
}

func MapDifference(iMapOld, iMapNew map[string]interface{}, uncheckedKeys []string) map[string]interface{} {
	differentFields := make(map[string]interface{})

	for oldMapK, oldMapV := range iMapOld {
		if contains(uncheckedKeys, oldMapK) {
			continue
		}

		if newMapV, ok := iMapNew[oldMapK]; !ok || !reflect.DeepEqual(newMapV, oldMapV) {
			differentFields[oldMapK] = oldMapV
		}
	}

	for newMapK, newMapV := range iMapNew {
		if contains(uncheckedKeys, newMapK) {
			continue
		}

		if _, ok := iMapOld[newMapK]; !ok {
			differentFields[newMapK] = newMapV
		}
	}

	return differentFields
}
