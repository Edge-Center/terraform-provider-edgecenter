//go:build integration

package edgemon_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func rmonResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}
