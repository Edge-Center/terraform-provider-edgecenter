package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func Provider() *schema.Provider {
	resources, dataSources := registerAll(edgecenter.LegacyService{})

	p := &schema.Provider{
		Schema:         edgecenter.ProviderSchema(),
		ResourcesMap:   resources,
		DataSourcesMap: dataSources,
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			terraformVersion = "0.12+compatible"
		}
		return edgecenter.ProviderConfigure(ctx, d, terraformVersion)
	}

	return p
}
