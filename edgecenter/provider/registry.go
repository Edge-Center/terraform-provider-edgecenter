package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Registrar interface {
	Name() string
	Resources() map[string]*schema.Resource
	DataSources() map[string]*schema.Resource
}

func registerAll(services ...Registrar) (map[string]*schema.Resource, map[string]*schema.Resource) {
	resources := make(map[string]*schema.Resource)
	dataSources := make(map[string]*schema.Resource)

	for _, svc := range services {
		for name, r := range svc.Resources() {
			if _, dup := resources[name]; dup {
				panic(fmt.Sprintf("resource %q registered by multiple services (last: %s)", name, svc.Name()))
			}
			resources[name] = r
		}
		for name, ds := range svc.DataSources() {
			if _, dup := dataSources[name]; dup {
				panic(fmt.Sprintf("data source %q registered by multiple services (last: %s)", name, svc.Name()))
			}
			dataSources[name] = ds
		}
	}

	return resources, dataSources
}
