//go:build integration

package reseller_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testEntityID    = 4242
	testEntityIDStr = "4242"
	testEntityType  = "reseller"

	testRegionID    = 8
	testRegionIDStr = "8"
)

func resellerResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func resellerDataSource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().DataSourcesMap[name]
	if res == nil {
		t.Fatalf("data source %q is not registered in the provider", name)
	}

	return res
}

func ptr[T any](value T) *T {
	return &value
}
