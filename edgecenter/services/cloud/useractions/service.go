package useractions

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cloud_useractions" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		SubscriptionAMQPResource: resourceUserActionsSubscriptionAMQP(),
		SubscriptionLogResource:  resourceUserActionsSubscriptionLog(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		SubscriptionAMQPDataSource: dataSourceUserActionsListAMQPSubscriptions(),
		SubscriptionLogDataSource:  dataSourceUserActionsListLogSubscriptions(),
	}
}
