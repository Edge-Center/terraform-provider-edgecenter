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
	aliasName    = "sub.protected.example.com"
	aliasNewName = "other.protected.example.com"
)

func aliasConfig(resource, name string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionAliasSchemaResource: resource,
		protectionsvc.ProtectionAliasSchemaName:     name,
	}
}

func aliasRemote(name string) *protectionSDK.Alias {
	return &protectionSDK.Alias{ID: childID, Name: name, SSLStatus: "not_issued"}
}

func aliasCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.AliasCreateRequest

	mc.Aliases.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.AliasCreateRequest) }).
		Return(aliasRemote(aliasName), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(aliasRemote(aliasName), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends only the name and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, aliasName, sent.Name)
			require.Nil(t, sent.SSLType)
			require.Nil(t, sent.SSLCrt)
			require.Nil(t, sent.SSLKey)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasSchemaResource: resourceIDStr,
				protectionsvc.ProtectionAliasSchemaName:     aliasName,
			})
		},
	}
}

func aliasCreatePaddedResourceCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(aliasRemote(aliasName), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(aliasRemote(aliasName), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "a zero padded resource reference is kept in the id but normalised in state",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasConfig("042", aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "042:7")
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasSchemaResource: resourceIDStr,
			})
		},
	}
}

func aliasCreateNonNumericResourceCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create rejects a resource reference that is not a number",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasConfig("protected.example.com", aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid syntax")
			fake.Aliases.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: alias must be a sub-domain of the resource"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "must be a sub-domain")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Aliases.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(aliasRemote(aliasName), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the composite id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func aliasReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(aliasRemote(aliasNewName), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read replaces the name with the one the api reports",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionAliasSchemaResource: resourceIDStr,
				protectionsvc.ProtectionAliasSchemaName:     aliasNewName,
			})
		},
	}
}

func aliasReadIgnoresCertificateFieldsCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	remote := aliasRemote(aliasName)
	remote.SSLType = ptr("custom")
	remote.SSLExpire = 1893456000
	remote.SSLStatus = "issued"

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read keeps the certificate fields out of state because the alias schema has none",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.NotContains(t, state.Attributes, "ssl_type")
			require.NotContains(t, state.Attributes, "ssl_status")
			require.NotContains(t, state.Attributes, "ssl_expire")
		},
	}
}

func aliasReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: alias doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "alias doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func aliasReadMalformedIDCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read rejects an id without a separator",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong input id")
			fake.Aliases.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasRenameReplacesCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)
	mc.Aliases.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(aliasRemote(aliasNewName), nil, nil)
	mc.Aliases.On("Get", mock.Anything, resourceID, childID).Return(aliasRemote(aliasNewName), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "renaming the alias destroys it and creates a new one",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		NewConfig:    aliasConfig(resourceIDStr, aliasNewName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
			fake.Aliases.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
			fake.Aliases.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func aliasDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Aliases.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
		},
	}
}

func aliasDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Aliases.On("Delete", mock.Anything, resourceID, childID).
		Return(nil, fmt.Errorf("api error: alias doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: aliasConfig(resourceIDStr, aliasName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "alias doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionAlias_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionAliasResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		aliasCreateCase(),
		aliasCreatePaddedResourceCase(),
		aliasCreateNonNumericResourceCase(),
		aliasCreateAPIFailureCase(),
		aliasCreateReadFailureCase(),
		aliasReadCase(),
		aliasReadIgnoresCertificateFieldsCase(),
		aliasReadAPIFailureCase(),
		aliasReadMalformedIDCase(),
		aliasRenameReplacesCase(),
		aliasDeleteCase(),
		aliasDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
