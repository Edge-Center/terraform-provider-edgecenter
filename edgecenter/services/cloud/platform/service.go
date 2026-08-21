package platform

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_platform" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		ProjectResource: resourceProject(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		AvailabilityZoneDataSource: dataSourceAvailabilityZone(),
		FlavorDataSource:           dataSourceFlavor(),
		ImageDataSource:            dataSourceImage(),
		ProjectDataSource:          dataSourceProject(),
		RegionDataSource:           dataSourceRegion(),
	}
}
