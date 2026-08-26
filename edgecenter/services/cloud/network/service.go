package network

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_network" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		FloatingIPResource:      resourceFloatingIP(),
		NetworkResource:         resourceNetwork(),
		ReservedFixedIPResource: resourceReservedFixedIP(),
		RouterResource:          resourceRouter(),
		SubnetResource:          resourceSubnet(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		FloatingIPDataSource:      dataSourceFloatingIP(),
		NetworkDataSource:         dataSourceNetwork(),
		ReservedFixedIPDataSource: dataSourceReservedFixedIP(),
		RouterDataSource:          dataSourceRouter(),
		SubnetDataSource:          dataSourceSubnet(),
	}
}
