//go:build integration

package storage_test

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func storageResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func storageDataSource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	ds := provider.Provider().DataSourcesMap[name]
	if ds == nil {
		t.Fatalf("data source %q is not registered in the provider", name)
	}

	return ds
}

func anyOpts(n int) []interface{} {
	args := make([]interface{}, n)
	for i := range args {
		args[i] = mock.Anything
	}

	return args
}

func appliedOpts[T any](args mock.Arguments) T {
	var params T
	for _, arg := range args {
		opt, ok := arg.(func(*T))
		if !ok {
			continue
		}
		opt(&params)
	}

	return params
}
