package dbaas

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "dbaas" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		DBaaSClusterResource:  resourceDBaaSCluster(),
		DBaaSDatabaseResource: resourceDBaaSDatabase(),
		DBaaSUserResource:     resourceDBaaSUser(),
		DBaaSBackupResource:   resourceDBaaSBackup(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		DBaaSDbmsDataSource:      dataSourceDBaaSDBMS(),
		DBaaSClustersDataSource:  dataSourceDBaaSClusters(),
		DBaaSDatabasesDataSource: dataSourceDBaaSDatabases(),
		DBaaSUsersDataSource:     dataSourceDBaaSUsers(),
		DBaaSBackupDataSource:    dataSourceDBaaSBackup(),
	}
}
