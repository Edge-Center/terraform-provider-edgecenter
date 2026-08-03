package protection

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

type Service struct{}

func (Service) Name() string { return "protection" }

func (Service) Resources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		ProtectionResourceResource:         resourceProtectionResource(),
		ProtectionCertificateResource:      resourceProtectionResourceCertificate(),
		ProtectionOriginResource:           resourceProtectionResourceOrigin(),
		ProtectionHeaderResource:           resourceProtectionResourceHeader(),
		ProtectionBlacklistEntryResource:   resourceProtectionResourceBlacklistEntry(),
		ProtectionWhitelistEntryResource:   resourceProtectionResourceWhitelistEntry(),
		ProtectionAliasResource:            resourceProtectionResourceAlias(),
		ProtectionAliasCertificateResource: resourceProtectionResourceAliasCertificate(),
	}
}

func (Service) DataSources() map[string]*schema.Resource { return nil }
