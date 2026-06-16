package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type stubService struct {
	name        string
	resources   map[string]*schema.Resource
	dataSources map[string]*schema.Resource
}

func (s stubService) Name() string                             { return s.name }
func (s stubService) Resources() map[string]*schema.Resource   { return s.resources }
func (s stubService) DataSources() map[string]*schema.Resource { return s.dataSources }

func TestRegisterAllMergesServices(t *testing.T) {
	t.Parallel()

	a := stubService{name: "a", resources: map[string]*schema.Resource{"edgecenter_a": {}}}
	b := stubService{name: "b", dataSources: map[string]*schema.Resource{"edgecenter_b": {}}}

	resources, dataSources := registerAll(a, b)

	if len(resources) != 1 || resources["edgecenter_a"] == nil {
		t.Fatalf("resources not merged: %v", resources)
	}
	if len(dataSources) != 1 || dataSources["edgecenter_b"] == nil {
		t.Fatalf("data sources not merged: %v", dataSources)
	}
}

func TestRegisterAllPanicsOnDuplicateResource(t *testing.T) {
	t.Parallel()

	a := stubService{name: "a", resources: map[string]*schema.Resource{"edgecenter_dup": {}}}
	b := stubService{name: "b", resources: map[string]*schema.Resource{"edgecenter_dup": {}}}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate resource name")
		}
		want := `resource "edgecenter_dup" registered by multiple services (last: b)`
		if got, ok := r.(string); !ok || got != want {
			t.Fatalf("panic = %v, want %q", r, want)
		}
	}()

	registerAll(a, b)
}
