package dns

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "dns" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		DNSZoneResource:          resourceDNSZone(),
		DNSZoneRecordResource:    resourceDNSZoneRecord(),
		DNSSecondaryZoneResource: resourceDNSSecondaryZone(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		DNSSecondaryZonesDataSource: dataSourceDNSSecondaryZones(),
	}
}
