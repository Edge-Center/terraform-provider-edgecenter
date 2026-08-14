package mkaas

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "mkaas" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		MKaaSClusterResource: resourceMKaaSCluster(),
		MKaaSPoolResource:    resourceMKaaSPool(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		MKaaSClusterDataSource:    dataSourceMKaaSCluster(),
		MKaaSPoolDataSource:       dataSourceMKaaSPool(),
		K8sDataSource:             dataSourceK8s(),
		K8sPoolDataSource:         dataSourceK8sPool(),
		K8sClientConfigDataSource: dataSourceK8sClientConfig(),
	}
}
