package websec

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "websec" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"edgecenter_websec_resource":        resourceWebsecResource(),
		"edgecenter_websec_source_group":    resourceWebsecSourceGroup(),
		"edgecenter_websec_ssl_certificate": resourceWebsecSSLCertificate(),
		"edgecenter_websec_location":        resourceWebsecLocation(),
		"edgecenter_websec_lecert":          ResourceWebsecLECert(leCertPollInterval),
	}
}

func (Service) DataSources() map[string]*schema.Resource { return nil }
