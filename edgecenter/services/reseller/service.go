package reseller

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "reseller" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		ResellerImagesResource:   resourceResellerImages(),
		ResellerImagesV2Resource: resourceResellerImagesV2(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		ResellerImagesDataSource:   dataSourceResellerImages(),
		ResellerNetworksDataSource: dataSourceResellerNetworksList(),
		ResellerImagesV2DataSource: dataSourceResellerImagesV2(),
	}
}
