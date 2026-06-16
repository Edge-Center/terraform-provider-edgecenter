package edgemon

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const RMONPlaceAll = "all"

type Service struct{}

func (Service) Name() string { return "edgemon" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"edgecenter_rmon_channel":        resourceRMONChannel(),
		"edgecenter_rmon_check_dns":      resourceRMONCheckDNS(),
		"edgecenter_rmon_check_group":    resourceRMONCheckGroup(),
		"edgecenter_rmon_check_http":     resourceRMONCheckHTTP(),
		"edgecenter_rmon_check_ping":     resourceRMONCheckPing(),
		"edgecenter_rmon_check_rabbitmq": resourceRMONCheckRabbitMQ(),
		"edgecenter_rmon_check_smtp":     resourceRMONCheckSMTP(),
		"edgecenter_rmon_check_tcp":      resourceRMONCheckTCP(),
		"edgecenter_rmon_status_page":    resourceRMONStatusPage(),
	}
}

func (Service) DataSources() map[string]*schema.Resource { return nil }
