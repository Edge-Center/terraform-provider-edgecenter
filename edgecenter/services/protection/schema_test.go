package protection

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistersEveryProtectionName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "protection", Service{}.Name())

	resources := Service{}.Resources()
	require.Len(t, resources, 8)
	for _, name := range []string{
		ProtectionResourceResource,
		ProtectionCertificateResource,
		ProtectionOriginResource,
		ProtectionHeaderResource,
		ProtectionBlacklistEntryResource,
		ProtectionWhitelistEntryResource,
		ProtectionAliasResource,
		ProtectionAliasCertificateResource,
	} {
		require.Containsf(t, resources, name, "resource %q is not registered", name)
	}

	require.Nil(t, Service{}.DataSources())
}

func TestEveryProtectionResourceIsImportable(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		require.NotNilf(t, res.Importer, "resource %q has no importer", name)
		require.NotNilf(t, res.CreateContext, "resource %q has no create", name)
		require.NotNilf(t, res.ReadContext, "resource %q has no read", name)
		require.NotNilf(t, res.DeleteContext, "resource %q has no delete", name)
	}
}

func TestProtectionResourceSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResource()

	name := res.Schema[ProtectionResourceSchemaName]
	require.True(t, name.Required)
	require.True(t, name.ForceNew)

	tls := res.Schema[ProtectionResourceSchemaTLS]
	require.True(t, tls.Required)
	require.Equal(t, schema.TypeSet, tls.Type)
	require.Equal(t, 1, tls.MinItems)

	for _, key := range []string{
		ProtectionResourceSchemaActive,
		ProtectionResourceSchemaGeoIPList,
		ProtectionResourceSchemaGeoIPMode,
		ProtectionResourceSchemaHTTPToOrigin,
		ProtectionResourceSchemaLoadBalancingType,
		ProtectionResourceSchemaMultipleOrigins,
		ProtectionResourceSchemaRedirectToHTTPS,
		ProtectionResourceSchemaWildcardAliases,
		ProtectionResourceSchemaWAF,
		ProtectionResourceSchemaWWWRedirect,
	} {
		attr := res.Schema[key]
		require.NotNilf(t, attr, "attribute %q is missing", key)
		require.Truef(t, attr.Optional, "attribute %q must stay optional", key)
		require.Truef(t, attr.Computed, "attribute %q must stay computed", key)
		require.Falsef(t, attr.ForceNew, "attribute %q must not force replacement", key)
	}

	for _, key := range []string{
		ProtectionResourceSchemaClient,
		ProtectionResourceSchemaEnabled,
		ProtectionResourceSchemaIP,
		ProtectionResourceSchemaStatus,
		ProtectionResourceSchemaWaitForLE,
	} {
		attr := res.Schema[key]
		require.NotNilf(t, attr, "attribute %q is missing", key)
		require.Truef(t, attr.Computed, "attribute %q must stay computed", key)
		require.Falsef(t, attr.Optional, "attribute %q must stay read only", key)
	}
}

func TestProtectionResourceGeoIPModeValidation(t *testing.T) {
	t.Parallel()

	validate := resourceProtectionResource().Schema[ProtectionResourceSchemaGeoIPMode].ValidateFunc
	require.NotNil(t, validate)

	for _, mode := range []string{geoIPNo, geoIPAllowList, geoIPBlockList} {
		_, errs := validate(mode, ProtectionResourceSchemaGeoIPMode)
		require.Emptyf(t, errs, "%q must be accepted", mode)
	}

	for _, mode := range []string{"", "No", "deny", "allowlist"} {
		_, errs := validate(mode, ProtectionResourceSchemaGeoIPMode)
		require.NotEmptyf(t, errs, "%q must be rejected", mode)
	}
}

func TestProtectionResourceLoadBalancingTypeValidation(t *testing.T) {
	t.Parallel()

	validate := resourceProtectionResource().Schema[ProtectionResourceSchemaLoadBalancingType].ValidateFunc
	require.NotNil(t, validate)

	for _, lb := range []string{lbRoundRobin, lbIPHash} {
		_, errs := validate(lb, ProtectionResourceSchemaLoadBalancingType)
		require.Emptyf(t, errs, "%q must be accepted", lb)
	}

	for _, lb := range []string{"", "round robin", "ip_hash"} {
		_, errs := validate(lb, ProtectionResourceSchemaLoadBalancingType)
		require.NotEmptyf(t, errs, "%q must be rejected", lb)
	}
}

func TestProtectionResourceTLSHasNoValidation(t *testing.T) {
	t.Parallel()

	tls := resourceProtectionResource().Schema[ProtectionResourceSchemaTLS]
	require.Nil(t, tls.ValidateFunc)
	require.Nil(t, tls.ValidateDiagFunc)

	elem, ok := tls.Elem.(*schema.Schema)
	require.True(t, ok)
	require.Nil(t, elem.ValidateFunc)
	require.Nil(t, elem.ValidateDiagFunc)
}

func TestProtectionCertificateSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResourceCertificate()

	require.NotNil(t, res.UpdateContext)

	resource := res.Schema[ProtectionCertificateSchemaResource]
	require.True(t, resource.Required)
	require.True(t, resource.ForceNew)

	sslType := res.Schema[ProtectionCertificateSchemaSSLType]
	require.True(t, sslType.Required)
	require.NotNil(t, sslType.ValidateFunc)

	require.True(t, res.Schema[ProtectionCertificateSchemaSSLKey].Sensitive)
	require.False(t, res.Schema[ProtectionCertificateSchemaSSLCrt].Sensitive)

	for _, key := range []string{
		ProtectionCertificateSchemaSSLExpire,
		ProtectionCertificateSchemaSSLStatus,
	} {
		require.Truef(t, res.Schema[key].Computed, "attribute %q must stay computed", key)
	}
}

func TestProtectionCertificateSSLTypeValidation(t *testing.T) {
	t.Parallel()

	validate := resourceProtectionResourceCertificate().Schema[ProtectionCertificateSchemaSSLType].ValidateFunc
	require.NotNil(t, validate)

	for _, sslType := range []string{sslCustom, sslLE} {
		_, errs := validate(sslType, ProtectionCertificateSchemaSSLType)
		require.Emptyf(t, errs, "%q must be accepted", sslType)
	}

	for _, sslType := range []string{"", "LE", "lets_encrypt"} {
		_, errs := validate(sslType, ProtectionCertificateSchemaSSLType)
		require.NotEmptyf(t, errs, "%q must be rejected", sslType)
	}
}

func TestProtectionOriginSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResourceOrigin()

	ip := res.Schema[ProtectionOriginSchemaIP]
	require.True(t, ip.Required)
	require.True(t, ip.ForceNew)
	require.Nil(t, ip.ValidateFunc)
	require.Nil(t, ip.ValidateDiagFunc)

	resource := res.Schema[ProtectionOriginSchemaResource]
	require.True(t, resource.Required)
	require.True(t, resource.ForceNew)

	for _, key := range []string{
		ProtectionOriginSchemaComment,
		ProtectionOriginSchemaFailTimeout,
		ProtectionOriginSchemaMaxFails,
		ProtectionOriginSchemaMode,
		ProtectionOriginSchemaWeight,
	} {
		attr := res.Schema[key]
		require.NotNilf(t, attr, "attribute %q is missing", key)
		require.Truef(t, attr.Optional, "attribute %q must stay optional", key)
		require.Truef(t, attr.Computed, "attribute %q must stay computed", key)
	}
}

func TestProtectionOriginModeValidation(t *testing.T) {
	t.Parallel()

	validate := resourceProtectionResourceOrigin().Schema[ProtectionOriginSchemaMode].ValidateFunc
	require.NotNil(t, validate)

	for _, mode := range []string{modePrimary, modeBackup, modeDown} {
		_, errs := validate(mode, ProtectionOriginSchemaMode)
		require.Emptyf(t, errs, "%q must be accepted", mode)
	}

	for _, mode := range []string{"", "Primary", "standby"} {
		_, errs := validate(mode, ProtectionOriginSchemaMode)
		require.NotEmptyf(t, errs, "%q must be rejected", mode)
	}
}

func TestProtectionHeaderSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResourceHeader()

	require.True(t, res.Schema[ProtectionHeaderSchemaKey].Required)
	require.False(t, res.Schema[ProtectionHeaderSchemaKey].ForceNew)
	require.True(t, res.Schema[ProtectionHeaderSchemaValue].Required)
	require.False(t, res.Schema[ProtectionHeaderSchemaValue].ForceNew)

	resource := res.Schema[ProtectionHeaderSchemaResource]
	require.True(t, resource.Required)
	require.True(t, resource.ForceNew)
}

func TestProtectionListEntrySchemas(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		res      *schema.Resource
		ip       string
		resource string
	}{
		ProtectionBlacklistEntryResource: {
			res:      resourceProtectionResourceBlacklistEntry(),
			ip:       ProtectionBlacklistEntrySchemaIP,
			resource: ProtectionBlacklistEntrySchemaResource,
		},
		ProtectionWhitelistEntryResource: {
			res:      resourceProtectionResourceWhitelistEntry(),
			ip:       ProtectionWhitelistEntrySchemaIP,
			resource: ProtectionWhitelistEntrySchemaResource,
		},
	}

	for name, tc := range cases {
		ip := tc.res.Schema[tc.ip]
		require.Truef(t, ip.Required, "%s: ip must be required", name)
		require.Falsef(t, ip.ForceNew, "%s: ip must be updated in place", name)
		require.Nilf(t, ip.ValidateFunc, "%s: ip has no validator", name)
		require.Nilf(t, ip.ValidateDiagFunc, "%s: ip has no validator", name)

		resource := tc.res.Schema[tc.resource]
		require.Truef(t, resource.Required, "%s: resource must be required", name)
		require.Truef(t, resource.ForceNew, "%s: resource must force replacement", name)
	}
}

func TestProtectionAliasSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResourceAlias()

	require.Nil(t, res.UpdateContext, "every attribute is ForceNew")

	for _, key := range []string{ProtectionAliasSchemaName, ProtectionAliasSchemaResource} {
		attr := res.Schema[key]
		require.NotNilf(t, attr, "attribute %q is missing", key)
		require.Truef(t, attr.Required, "attribute %q must stay required", key)
		require.Truef(t, attr.ForceNew, "attribute %q must force replacement", key)
	}
}

func TestProtectionAliasCertificateSchema(t *testing.T) {
	t.Parallel()

	res := resourceProtectionResourceAliasCertificate()

	require.NotNil(t, res.UpdateContext)

	alias := res.Schema[ProtectionAliasCertificateSchemaAlias]
	require.True(t, alias.Required)
	require.True(t, alias.ForceNew)

	require.True(t, res.Schema[ProtectionAliasCertificateSchemaSSLKey].Sensitive)
	require.False(t, res.Schema[ProtectionAliasCertificateSchemaSSLCrt].Sensitive)

	for _, key := range []string{
		ProtectionAliasCertificateSchemaSSLExpire,
		ProtectionAliasCertificateSchemaSSLStatus,
	} {
		require.Truef(t, res.Schema[key].Computed, "attribute %q must stay computed", key)
	}
}

func TestProtectionAliasCertificateSSLTypeValidation(t *testing.T) {
	t.Parallel()

	validate := resourceProtectionResourceAliasCertificate().Schema[ProtectionAliasCertificateSchemaSSLType].ValidateFunc
	require.NotNil(t, validate)

	for _, sslType := range []string{sslCustom, sslLE} {
		_, errs := validate(sslType, ProtectionAliasCertificateSchemaSSLType)
		require.Emptyf(t, errs, "%q must be accepted", sslType)
	}

	for _, sslType := range []string{"", "custom "} {
		_, errs := validate(sslType, ProtectionAliasCertificateSchemaSSLType)
		require.NotEmptyf(t, errs, "%q must be rejected", sslType)
	}
}
