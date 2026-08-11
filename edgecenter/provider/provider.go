package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/cdn"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/edgemon"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/protection"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/reseller"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/storage"
)

func Provider() *schema.Provider {
	resources, dataSources := registerAll(
		edgecenter.LegacyService{},
		edgemon.Service{},
		cdn.Service{},
		dns.Service{},
		storage.Service{},
		protection.Service{},
		reseller.Service{},
		dbaas.Service{},
	)

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
