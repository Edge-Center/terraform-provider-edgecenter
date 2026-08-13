package storage

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "storage" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		StorageS3Resource:       resourceStorageS3(),
		StorageS3BucketResource: resourceStorageS3Bucket(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		StorageS3DataSource:       dataSourceStorageS3(),
		StorageS3BucketDataSource: dataSourceStorageS3Bucket(),
	}
}
