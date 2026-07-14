package cdn

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "cdn" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"edgecenter_cdn_resource":    resourceCDNResource(),
		"edgecenter_cdn_origingroup": resourceCDNOriginGroup(),
		"edgecenter_cdn_lecert":      resourceCDNLECert(),
		"edgecenter_cdn_rule":        resourceCDNRule(),
		"edgecenter_cdn_shielding":   resourceCDNShielding(),
		"edgecenter_cdn_sslcert":     resourceCDNCert(),
	}
}

func (Service) DataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"edgecenter_cdn_client_info":        dataSourceCDNClientInfo(),
		"edgecenter_cdn_shielding_location": dataShieldingLocation(),
	}
}
