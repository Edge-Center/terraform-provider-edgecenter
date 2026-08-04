//go:build integration

package protection_test

import (
	"encoding/json"
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

func aliasCertConfig(alias, sslType string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionAliasCertificateSchemaAlias:   alias,
		protectionsvc.ProtectionAliasCertificateSchemaSSLType: sslType,
	}
}

func aliasCertCustomConfig(alias string) map[string]interface{} {
	config := aliasCertConfig(alias, "custom")
	config[protectionsvc.ProtectionAliasCertificateSchemaSSLCrt] = certBody
	config[protectionsvc.ProtectionAliasCertificateSchemaSSLKey] = certKey

	return config
}

func aliasCertRemote(sslType *string) *protectionSDK.Alias {
	return &protectionSDK.Alias{
		ID:        childID,
		Name:      aliasName,
		SSLType:   sslType,
		SSLExpire: certExpire,
		SSLStatus: certStatus,
	}
}

func aliasCertCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.AliasUpdateRequest

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.AliasUpdateRequest) }).
		Return(aliasCertRemote(ptr("custom")), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(aliasCertRemote(ptr("custom")), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends the certificate material and nothing else",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.NotNil(t, sent.SSLType)
			require.Equal(t, "custom", *sent.SSLType)
			require.NotNil(t, sent.SSLCrt)
			require.Equal(t, certBody, *sent.SSLCrt)
			require.NotNil(t, sent.SSLKey)
			require.Equal(t, certKey, *sent.SSLKey)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasCertificateSchemaAlias:     compositeID,
				protectionsvc.ProtectionAliasCertificateSchemaSSLType:   "custom",
				protectionsvc.ProtectionAliasCertificateSchemaSSLStatus: certStatus,
			})
		},
	}
}

func aliasCertCreateLetsEncryptCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.AliasUpdateRequest

	remote := aliasCertRemote(ptr("le"))
	remote.SSLStatus = "in_progress"

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.AliasUpdateRequest) }).
		Return(remote, nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "an acme certificate is requested without any certificate material and apply returns before issuance finishes",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertConfig(compositeID, "le"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, "le", *sent.SSLType)
			require.Nil(t, sent.SSLCrt)
			require.Nil(t, sent.SSLKey)

			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasCertificateSchemaSSLStatus: "in_progress",
			})
		},
	}
}

func aliasCertCreateNormalisesAliasCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(aliasCertRemote(ptr("custom")), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(aliasCertRemote(ptr("custom")), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "a zero padded alias reference is rewritten in canonical form in both the id and the state",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertCustomConfig("042:007"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasCertificateSchemaAlias: compositeID,
			})
		},
	}
}

func aliasCertCreateMissingMaterialCase(name, drop, wantErr string) support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	config := aliasCertCustomConfig(compositeID)
	delete(config, drop)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      name,
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, wantErr)
			fake.Aliases.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasCertCreateMalformedAliasCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "an alias reference without a separator is rejected only at apply time",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertCustomConfig(resourceIDStr),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong input id")
			fake.Aliases.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasCertCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: certificate does not cover the alias"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "does not cover the alias")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Aliases.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasCertCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(aliasCertRemote(ptr("custom")), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func aliasCertReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(aliasCertRemote(ptr("le")), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read fills the certificate status fields and rewrites the alias reference",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasCertificateSchemaAlias:     compositeID,
				protectionsvc.ProtectionAliasCertificateSchemaSSLType:   "le",
				protectionsvc.ProtectionAliasCertificateSchemaSSLStatus: certStatus,
				protectionsvc.ProtectionAliasCertificateSchemaSSLExpire: "1893456000",
			})
		},
	}
}

func aliasCertReadKeepsMaterialFromStateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(aliasCertRemote(ptr("custom")), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read never refreshes the certificate or the private key, they stay whatever state already held",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasCertificateSchemaSSLCrt: certBody,
				protectionsvc.ProtectionAliasCertificateSchemaSSLKey: certKey,
			})
		},
	}
}

func aliasCertReadNullTypeCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	remote := aliasCertRemote(nil)
	remote.SSLStatus = ""
	remote.SSLExpire = 0

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "an alias without a certificate empties ssl_type in state although the attribute is required",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Empty(t, state.Attributes[protectionsvc.ProtectionAliasCertificateSchemaSSLType])
		},
	}
}

func aliasCertReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: alias doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "alias doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func aliasCertDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.AliasUpdateRequest

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.AliasUpdateRequest) }).
		Return(aliasCertRemote(nil), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete sends an empty update that carries no certificate material at all",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)

			require.NotNil(t, sent)
			require.Nil(t, sent.SSLType)
			require.Nil(t, sent.SSLCrt)
			require.Nil(t, sent.SSLKey)

			body, err := json.Marshal(sent)
			require.NoError(t, err)
			require.JSONEq(t, `{"alias_ssl_type":null}`, string(body))
		},
	}
}

func aliasCertDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: alias doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasCertCustomConfig(compositeID),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "alias doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionAliasCertificate_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionAliasCertificateResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		aliasCertCreateCase(),
		aliasCertCreateLetsEncryptCase(),
		aliasCertCreateNormalisesAliasCase(),
		aliasCertCreateMissingMaterialCase(
			"a custom certificate without the public part fails at apply time",
			protectionsvc.ProtectionAliasCertificateSchemaSSLCrt,
			"No certificate set for "+resourceIDStr),
		aliasCertCreateMissingMaterialCase(
			"a custom certificate without the private key fails at apply time",
			protectionsvc.ProtectionAliasCertificateSchemaSSLKey,
			"No certificate key set for "+resourceIDStr),
		aliasCertCreateMalformedAliasCase(),
		aliasCertCreateAPIFailureCase(),
		aliasCertCreateReadFailureCase(),
		aliasCertReadCase(),
		aliasCertReadKeepsMaterialFromStateCase(),
		aliasCertReadNullTypeCase(),
		aliasCertReadAPIFailureCase(),
		aliasCertDeleteCase(),
		aliasCertDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
