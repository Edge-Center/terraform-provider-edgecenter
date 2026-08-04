//go:build integration

package protection_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	protectionmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/protection/mock"
	protectionsvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/protection"
)

const (
	certBody   = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"
	certKey    = "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----"
	certExpire = 1893456000
	certStatus = "issued"
)

func certConfig(sslType string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionCertificateSchemaResource: resourceIDStr,
		protectionsvc.ProtectionCertificateSchemaSSLType:  sslType,
	}
}

func certCustomConfig() map[string]interface{} {
	config := certConfig("custom")
	config[protectionsvc.ProtectionCertificateSchemaSSLCrt] = certBody
	config[protectionsvc.ProtectionCertificateSchemaSSLKey] = certKey

	return config
}

func certParent() *protectionSDK.Resource {
	return &protectionSDK.Resource{
		ID:              resourceID,
		Name:            "protected.example.com",
		Active:          true,
		WAF:             true,
		RedirectToHTTPS: true,
		HTTPS2HTTP:      1,
		IPHash:          1,
		GeoIPMode:       2,
		GeoIPList:       "RU,DE",
		WWWRedir:        1,
		MultipleOrigins: true,
		WidlcardAliases: true,
		TLSEnabled:      []string{"1.2", "1.3"},
		SSLType:         ptr("custom"),
		SSLExpire:       certExpire,
		SSLStatus:       certStatus,
	}
}

func certCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceUpdateRequest

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.ResourceUpdateRequest) }).
		Return(certParent(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create carries every parent setting forward and adds the certificate material",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.True(t, sent.Active)
			require.True(t, sent.WAF)
			require.True(t, sent.RedirectToHTTPS)
			require.True(t, sent.MultipleOrigins)
			require.True(t, sent.WidlcardAliases)
			require.Equal(t, byte(1), sent.HTTPS2HTTP)
			require.Equal(t, byte(1), sent.IPHash)
			require.Equal(t, byte(2), sent.GeoIPMode)
			require.Equal(t, "RU,DE", sent.GeoIPList)
			require.Equal(t, byte(1), sent.WWWRedir)
			require.Equal(t, []string{"1.2", "1.3"}, sent.TLSEnabled)

			require.NotNil(t, sent.SSLType)
			require.Equal(t, "custom", *sent.SSLType)
			require.NotNil(t, sent.SSLCert)
			require.Equal(t, certBody, *sent.SSLCert)
			require.NotNil(t, sent.SSLKey)
			require.Equal(t, certKey, *sent.SSLKey)

			support.RequireStateID(t, state, resourceIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionCertificateSchemaResource:  resourceIDStr,
				protectionsvc.ProtectionCertificateSchemaSSLType:   "custom",
				protectionsvc.ProtectionCertificateSchemaSSLStatus: certStatus,
			})
		},
	}
}

func certCreateLetsEncryptCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceUpdateRequest

	parent := certParent()
	parent.SSLType = ptr("le")
	parent.SSLStatus = "in_progress"
	parent.SSLExpire = 0

	mc.Resources.On("Get", mock.Anything, resourceID).Return(parent, nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.ResourceUpdateRequest) }).
		Return(parent, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "an acme certificate is requested without any certificate material and apply returns before issuance finishes",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: certConfig("le"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.NotNil(t, sent.SSLType)
			require.Equal(t, "le", *sent.SSLType)
			require.Nil(t, sent.SSLCert)
			require.Nil(t, sent.SSLKey)

			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionCertificateSchemaSSLStatus: "in_progress",
			})
		},
	}
}

func certCreateIgnoresMaterialForLetsEncryptCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceUpdateRequest

	parent := certParent()
	parent.SSLType = ptr("le")

	mc.Resources.On("Get", mock.Anything, resourceID).Return(parent, nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.ResourceUpdateRequest) }).
		Return(parent, nil, nil)

	config := certConfig("le")
	config[protectionsvc.ProtectionCertificateSchemaSSLCrt] = certBody
	config[protectionsvc.ProtectionCertificateSchemaSSLKey] = certKey

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "certificate material written next to the acme type is dropped without a warning",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, sent.SSLCert)
			require.Nil(t, sent.SSLKey)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionCertificateSchemaSSLCrt: certBody,
			})
		},
	}
}

func certCreateMissingMaterialCase(name, drop, wantErr string) support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)

	config := certCustomConfig()
	delete(config, drop)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      name,
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, wantErr)
			fake.Resources.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func certCreateGetFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).
		Return(nil, nil, fmt.Errorf("api error: resource doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error of the read that precedes the write",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "resource doesn't exist")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Resources.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func certCreateUpdateFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: certificate and key do not match"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error of the write",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "certificate and key do not match")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func certCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil).Once()
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).Return(certParent(), nil, nil)
	mc.Resources.On("Get", mock.Anything, resourceID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func certReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read fills the certificate status fields from the protected resource",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, resourceIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionCertificateSchemaResource:  resourceIDStr,
				protectionsvc.ProtectionCertificateSchemaSSLType:   "custom",
				protectionsvc.ProtectionCertificateSchemaSSLStatus: certStatus,
				protectionsvc.ProtectionCertificateSchemaSSLExpire: "1893456000",
			})
		},
	}
}

func certReadKeepsMaterialFromStateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read never refreshes the certificate or the private key, they stay whatever state already held",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionCertificateSchemaSSLCrt: certBody,
				protectionsvc.ProtectionCertificateSchemaSSLKey: certKey,
			})
		},
	}
}

func certReadNullTypeCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	parent := certParent()
	parent.SSLType = nil
	parent.SSLStatus = ""
	parent.SSLExpire = 0

	mc.Resources.On("Get", mock.Anything, resourceID).Return(parent, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "a resource without a certificate empties ssl_type in state although the attribute is required",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, resourceIDStr)
			require.Empty(t, state.Attributes[protectionsvc.ProtectionCertificateSchemaSSLType])
		},
	}
}

func certReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).
		Return(nil, nil, fmt.Errorf("api error: resource doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "resource doesn't exist")
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func certDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceUpdateRequest

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.ResourceUpdateRequest) }).
		Return(certParent(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete clears the certificate type and keeps every other parent setting",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)

			require.NotNil(t, sent)
			require.Nil(t, sent.SSLType)
			require.Nil(t, sent.SSLCert)
			require.Nil(t, sent.SSLKey)
			require.True(t, sent.Active)
			require.Equal(t, []string{"1.2", "1.3"}, sent.TLSEnabled)
		},
	}
}

func certDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(certParent(), nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: certificate is in use"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: certCustomConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "certificate is in use")
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func TestIntegrationProtectionCertificate_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionCertificateResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		certCreateCase(),
		certCreateLetsEncryptCase(),
		certCreateIgnoresMaterialForLetsEncryptCase(),
		certCreateMissingMaterialCase(
			"a custom certificate without the public part fails at apply time",
			protectionsvc.ProtectionCertificateSchemaSSLCrt,
			"No certificate set for "+resourceIDStr),
		certCreateMissingMaterialCase(
			"a custom certificate without the private key fails at apply time",
			protectionsvc.ProtectionCertificateSchemaSSLKey,
			"No certificate key set for "+resourceIDStr),
		certCreateGetFailureCase(),
		certCreateUpdateFailureCase(),
		certCreateReadFailureCase(),
		certReadCase(),
		certReadKeepsMaterialFromStateCase(),
		certReadNullTypeCase(),
		certReadAPIFailureCase(),
		certDeleteCase(),
		certDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
