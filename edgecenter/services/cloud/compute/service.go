package compute

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_compute" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		BaremetalResource:   resourceBmInstance(),
		InstanceResource:    resourceInstance(),
		InstanceV2Resource:  resourceInstanceV2(),
		KeypairResource:     resourceKeypair(),
		ServerGroupResource: resourceServerGroup(),
		SnapshotResource:    resourceSnapshot(),
		VolumeResource:      resourceVolume(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		InstanceDataSource:    dataSourceInstance(),
		InstanceV2DataSource:  dataSourceInstanceV2(),
		ServerGroupDataSource: dataSourceServerGroup(),
		SnapshotDataSource:    dataSourceSnapshot(),
		VolumeDataSource:      dataSourceVolume(),
	}
}
