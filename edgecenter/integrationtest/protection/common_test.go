//go:build integration

package protection_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	resourceID    int64 = 42
	resourceIDStr       = "42"

	childID    int64 = 7
	childIDStr       = "7"

	compositeID = resourceIDStr + ":" + childIDStr
)

func protectionResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func ptr[T any](value T) *T {
	return &value
}

func splitList(value string) []string {
	return strings.Split(value, ",")
}

func setValues(state *terraform.InstanceState, key string) []string {
	values := make([]string, 0)
	for attr, value := range state.Attributes {
		if strings.HasPrefix(attr, key+".") && attr != key+".#" {
			values = append(values, value)
		}
	}

	return values
}
