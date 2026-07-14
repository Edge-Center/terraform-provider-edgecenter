package cdn

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cdn "github.com/Edge-Center/edgecentercdn-go/edgecenter"
)

// fillValue populates every field of an option struct with a non-zero value, so a
// mapper that drops or mangles a field cannot hide behind a zero value.
func fillValue(t *testing.T, v reflect.Value, name string) {
	t.Helper()

	switch v.Kind() { //nolint:exhaustive // the default case fails the test on any kind the option structs do not use
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString(stringFor(name))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(intFor(name))
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		fillValue(t, v.Elem(), name)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fillValue(t, v.Field(i), v.Type().Field(i).Name)
		}
	case reflect.Slice:
		elem := v.Type().Elem()
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < 2; i++ {
			fillValue(t, s.Index(i), name)
			if elem.Kind() == reflect.String {
				s.Index(i).SetString(fmt.Sprintf("%s-%d", stringFor(name), i))
			}
			if elem.Kind() == reflect.Int {
				s.Index(i).SetInt(intFor(name) + int64(i))
			}
		}
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		val := reflect.New(v.Type().Elem()).Elem()
		fillValue(t, val, name)
		m.SetMapIndex(reflect.ValueOf("US"), val)
		v.Set(m)
	default:
		t.Fatalf("fillValue: unsupported kind %s for %s", v.Kind(), name)
	}
}

func stringFor(name string) string {
	switch name {
	case "PolicyType", "Mode":
		return "allow"
	case "Default":
		return "allow"
	case "Flag":
		return "break"
	case "SNIType":
		return "custom"
	case "Type":
		return "1"
	case "LimitType":
		return "static"
	default:
		return "v-" + name
	}
}

func intFor(name string) int64 {
	switch name {
	case "Code", "Codes":
		return 301
	case "Quality":
		return 80
	default:
		return 7
	}
}

// normalize makes two option trees comparable: sets do not preserve order, so every
// list is sorted before the structs are compared.
func normalize(t *testing.T, v interface{}) interface{} {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	return sortLists(out)
}

func sortLists(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		for key, val := range typed {
			typed[key] = sortLists(val)
		}

		return typed
	case []interface{}:
		for i := range typed {
			typed[i] = sortLists(typed[i])
		}
		sort.Slice(typed, func(i, j int) bool {
			left, _ := json.Marshal(typed[i])
			right, _ := json.Marshal(typed[j])

			return string(left) < string(right)
		})

		return typed
	default:
		return v
	}
}

func optionNames(t *testing.T, options interface{}) []string {
	t.Helper()

	raw, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func TestEveryResourceOptionSurvivesTheSchemaRoundTrip(t *testing.T) {
	t.Parallel()

	var want cdn.ResourceOptions
	fillValue(t, reflect.ValueOf(&want).Elem(), "ResourceOptions")

	names := optionNames(t, &want)
	if len(names) != 37 {
		t.Fatalf("expected 37 resource options, got %d: %v", len(names), names)
	}

	data := schema.TestResourceDataRaw(t, resourceCDNResource().Schema, map[string]interface{}{})
	if err := data.Set("options", resourceOptionsToList(&want)); err != nil {
		t.Fatalf("d.Set(options): %v", err)
	}

	got := listToResourceOptions(data.Get("options").([]interface{}))
	if got == nil {
		t.Fatal("listToResourceOptions returned nil for a fully populated options block")
	}

	requireSameOptions(t, &want, got, names)
}

func TestEveryLocationOptionSurvivesTheSchemaRoundTrip(t *testing.T) {
	t.Parallel()

	var want cdn.LocationOptions
	fillValue(t, reflect.ValueOf(&want).Elem(), "LocationOptions")

	names := optionNames(t, &want)
	if len(names) != 34 {
		t.Fatalf("expected 34 location options, got %d: %v", len(names), names)
	}

	data := schema.TestResourceDataRaw(t, resourceCDNRule().Schema, map[string]interface{}{})
	if err := data.Set("options", locationOptionsToList(&want)); err != nil {
		t.Fatalf("d.Set(options): %v", err)
	}

	got := listToLocationOptions(data.Get("options").([]interface{}))
	if got == nil {
		t.Fatal("listToLocationOptions returned nil for a fully populated options block")
	}

	requireSameOptions(t, &want, got, names)
}

func requireSameOptions(t *testing.T, want, got interface{}, names []string) {
	t.Helper()

	wantMap, _ := normalize(t, want).(map[string]interface{})
	gotMap, _ := normalize(t, got).(map[string]interface{})

	for _, name := range names {
		// without this the whole test is vacuous: a nil option compares equal to a nil option
		if wantMap[name] == nil {
			t.Fatalf("option %q was not populated by the filler, the comparison would be vacuous", name)
		}

		if !reflect.DeepEqual(wantMap[name], gotMap[name]) {
			t.Errorf("option %q did not survive the round trip:\n  sent to state: %#v\n  read back:     %#v",
				name, wantMap[name], gotMap[name])
		}
	}
}
