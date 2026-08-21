package lb

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_lb" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		L7PolicyResource:       resourceL7Policy(),
		L7RuleResource:         resourceL7Rule(),
		ListenerResource:       resourceLbListener(),
		LoadBalancerResource:   resourceLoadBalancer(),
		LoadBalancerV2Resource: resourceLoadBalancerV2(),
		MemberResource:         resourceLBMember(),
		PoolResource:           resourceLBPool(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		L7PolicyDataSource:       dataSourceL7Policy(),
		L7RuleDataSource:         datasourceL7Rule(),
		ListenerDataSource:       dataSourceLBListener(),
		LoadBalancerDataSource:   dataSourceLoadBalancer(),
		LoadBalancerV2DataSource: dataSourceLoadBalancerV2(),
		PoolDataSource:           dataSourceLBPool(),
	}
}
