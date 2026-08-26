package security

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_security" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		InstancePortSecurityResource: resourceInstancePortSecurity(),
		LifecyclePolicyResource:      resourceLifecyclePolicy(),
		SecretResource:               resourceSecret(),
		SecurityGroupResource:        resourceSecurityGroup(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		InstancePortSecurityDataSource: dataSourceInstancePortSecurity(),
		SecretDataSource:               dataSourceSecret(),
		SecurityGroupDataSource:        dataSourceSecurityGroup(),
	}
}
