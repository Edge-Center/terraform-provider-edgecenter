//go:build integration

package cdn_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func cdnResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func cdnDataSource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	ds := provider.Provider().DataSourcesMap[name]
	if ds == nil {
		t.Fatalf("data source %q is not registered in the provider", name)
	}

	return ds
}
