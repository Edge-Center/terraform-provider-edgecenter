package edgecenter

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// nolint: unused
func normalizeMetadata(metadata interface{}, defaults ...bool) (map[string]interface{}, error) {
	normalizedMetadata := map[string]interface{}{}
	readOnly := false

	if len(defaults) > 0 {
		readOnly = defaults[0]
	}

	switch metadata := metadata.(type) {
	default:
		return nil, fmt.Errorf("unexpected type %T", metadata)
	case []map[string]interface{}:
		for _, v := range metadata {
			normalizedMetadata[v["key"].(string)] = v
		}
	case map[string]interface{}:
		for k, v := range metadata {
			normalizedMetadata[k] = map[string]interface{}{
				"key":       k,
				"value":     v,
				"read_only": readOnly,
			}
		}
	case map[string]string:
		for k, v := range metadata {
			normalizedMetadata[k] = map[string]interface{}{
				"key":       k,
				"value":     v,
				"read_only": readOnly,
			}
		}
	}

	return normalizedMetadata, nil
}

// nolint: unused
func modulePrimaryInstanceState(ms *terraform.ModuleState, name string) (*terraform.InstanceState, error) {
	rs, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("not found: %s in %s", name, ms.Path)
	}

	is := rs.Primary
	if is == nil {
		return nil, fmt.Errorf("no primary instance: %s in %s", name, ms.Path)
	}

	return is, nil
}

// nolint: unused
func getMetadataFromResourceAttributes(prefix string, attributes *map[string]string) ([]map[string]interface{}, error) {
	metadataLength, err := strconv.Atoi((*attributes)[prefix+".#"])
	if err != nil {
		return nil, err
	}
	metadata := make([]map[string]interface{}, metadataLength)
	buildKey := func(idx int, name string) string {
		return fmt.Sprintf("%v.%v.%v", prefix, idx, name)
	}

	for i := 0; i < metadataLength; i++ {
		readOnly, err := strconv.ParseBool((*attributes)[buildKey(i, "read_only")])
		if err != nil {
			return nil, err
		}
		metadata[i] = map[string]interface{}{
			"key":       (*attributes)[buildKey(i, "key")],
			"value":     (*attributes)[buildKey(i, "value")],
			"read_only": readOnly,
		}
	}

	return metadata, nil
}

// nolint: unused
func checkMapInMap(srcMap map[string]interface{}, dstMap map[string]interface{}) bool {
	if len(srcMap) > len(dstMap) {
		return false
	}
	if len(srcMap) == len(dstMap) {
		return reflect.DeepEqual(srcMap, dstMap)
	}
	slicedMap := make(map[string]interface{}, len(srcMap))

	for k := range srcMap {
		if val, ok := dstMap[k]; ok {
			slicedMap[k] = val
		} else {
			return false
		}
	}

	return reflect.DeepEqual(srcMap, slicedMap)
}

// nolint: unused
func testAccCheckMetadata(name string, isMetaExists bool, metadataForCheck interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the resource by name from state
		ms := s.RootModule()
		is, err := modulePrimaryInstanceState(ms, name)
		if err != nil {
			return err
		}

		instanceMetadata, err := getMetadataFromResourceAttributes("metadata_read_only", &is.Attributes)
		if err != nil {
			return err
		}

		mt1, err := normalizeMetadata(metadataForCheck)
		if err != nil {
			return err
		}

		mt2, err := normalizeMetadata(instanceMetadata)
		if err != nil {
			return err
		}

		if !(checkMapInMap(mt1, mt2) == isMetaExists) {
			return fmt.Errorf("metadata not exist")
		}

		return nil
	}
}
