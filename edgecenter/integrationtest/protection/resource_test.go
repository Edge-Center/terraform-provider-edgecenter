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
	ddosName        = "site.example.com"
	ddosGeoIPList   = "RU,DE"
	ddosServiceIP   = "10.0.0.1"
	ddosStatus      = "ok"
	ddosLBIPHash    = "Round Robin with session persistence"
	ddosGeoIPBlock  = "block"
	ddosClientID    = 1188013
	ddosClientIDStr = "1188013"
)

func ddosConfig() map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionResourceSchemaName:              ddosName,
		protectionsvc.ProtectionResourceSchemaTLS:               []interface{}{"1.2", "1.3"},
		protectionsvc.ProtectionResourceSchemaActive:            true,
		protectionsvc.ProtectionResourceSchemaWAF:               true,
		protectionsvc.ProtectionResourceSchemaMultipleOrigins:   true,
		protectionsvc.ProtectionResourceSchemaWildcardAliases:   true,
		protectionsvc.ProtectionResourceSchemaRedirectToHTTPS:   true,
		protectionsvc.ProtectionResourceSchemaHTTPToOrigin:      true,
		protectionsvc.ProtectionResourceSchemaWWWRedirect:       true,
		protectionsvc.ProtectionResourceSchemaLoadBalancingType: ddosLBIPHash,
		protectionsvc.ProtectionResourceSchemaGeoIPMode:         ddosGeoIPBlock,
		protectionsvc.ProtectionResourceSchemaGeoIPList:         []interface{}{"RU", "DE"},
	}
}

func ddosRemote() *protectionSDK.Resource {
	return &protectionSDK.Resource{
		ID:              resourceID,
		Name:            ddosName,
		ClientID:        ddosClientID,
		Active:          true,
		Enabled:         true,
		WAF:             true,
		RedirectToHTTPS: true,
		Status:          ddosStatus,
		ServiceIP:       ddosServiceIP,
		HTTPS2HTTP:      1,
		IPHash:          1,
		GeoIPMode:       2,
		GeoIPList:       ddosGeoIPList,
		WWWRedir:        1,
		MultipleOrigins: true,
		WidlcardAliases: true,
		SSLType:         ptr("custom"),
		TLSEnabled:      []string{"1.2", "1.3"},
		WaitForLE:       600,
	}
}

func ddosCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceCreateRequest

	mc.Resources.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(1).(*protectionSDK.ResourceCreateRequest) }).
		Return(ddosRemote(), nil, nil)
	mc.Resources.On("Get", mock.Anything, resourceID).Return(ddosRemote(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create maps the booleans onto the numeric api fields and stores the numeric id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, ddosName, sent.Name)
			require.ElementsMatch(t, []string{"1.2", "1.3"}, sent.TLSEnabled)
			require.True(t, sent.Active)
			require.True(t, sent.WAF)
			require.True(t, sent.MultipleOrigins)
			require.True(t, sent.WidlcardAliases)
			require.True(t, sent.RedirectToHTTPS)
			require.Equal(t, byte(1), sent.HTTPS2HTTP)
			require.Equal(t, byte(1), sent.IPHash)
			require.Equal(t, byte(2), sent.GeoIPMode)
			require.ElementsMatch(t, []string{"RU", "DE"}, splitList(sent.GeoIPList))
			require.Equal(t, byte(1), sent.WWWRedir)
			require.Nil(t, sent.SSLType)

			support.RequireStateID(t, state, resourceIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionResourceSchemaName:   ddosName,
				protectionsvc.ProtectionResourceSchemaIP:     ddosServiceIP,
				protectionsvc.ProtectionResourceSchemaStatus: ddosStatus,
			})
		},
	}
}

func ddosCreateUnsetOptionsCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceCreateRequest

	remote := ddosRemote()
	remote.Active = false
	remote.WAF = false
	remote.MultipleOrigins = false
	remote.WidlcardAliases = false
	remote.RedirectToHTTPS = false
	remote.HTTPS2HTTP = 0
	remote.IPHash = 0
	remote.GeoIPMode = 0
	remote.GeoIPList = ""
	remote.WWWRedir = 0

	mc.Resources.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(1).(*protectionSDK.ResourceCreateRequest) }).
		Return(remote, nil, nil)
	mc.Resources.On("Get", mock.Anything, resourceID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:    "create sends every omitted option as its zero value",
		Op:      support.OpApply,
		Prepare: func() *protectionmock.MockedProtection { return mc },
		NewConfig: map[string]interface{}{
			protectionsvc.ProtectionResourceSchemaName: ddosName,
			protectionsvc.ProtectionResourceSchemaTLS:  []interface{}{"1.2"},
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.False(t, sent.Active)
			require.False(t, sent.WAF)
			require.Equal(t, byte(0), sent.HTTPS2HTTP)
			require.Equal(t, byte(0), sent.IPHash)
			require.Equal(t, byte(0), sent.GeoIPMode)
			require.Empty(t, sent.GeoIPList)

			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionResourceSchemaActive:            "false",
				protectionsvc.ProtectionResourceSchemaLoadBalancingType: "Round Robin",
				protectionsvc.ProtectionResourceSchemaGeoIPMode:         "no",
			})
		},
	}
}

func ddosCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Create", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: name is already taken"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "name is already taken")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Resources.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
		},
	}
}

func ddosCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Create", mock.Anything, mock.Anything).Return(ddosRemote(), nil, nil)
	mc.Resources.On("Get", mock.Anything, resourceID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func ddosReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(ddosRemote(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read maps the numeric api fields back onto the schema",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, resourceIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionResourceSchemaName:              ddosName,
				protectionsvc.ProtectionResourceSchemaActive:            "true",
				protectionsvc.ProtectionResourceSchemaWAF:               "true",
				protectionsvc.ProtectionResourceSchemaHTTPToOrigin:      "true",
				protectionsvc.ProtectionResourceSchemaWWWRedirect:       "true",
				protectionsvc.ProtectionResourceSchemaLoadBalancingType: ddosLBIPHash,
				protectionsvc.ProtectionResourceSchemaGeoIPMode:         ddosGeoIPBlock,
				protectionsvc.ProtectionResourceSchemaGeoIPList + ".#":  "2",
				protectionsvc.ProtectionResourceSchemaTLS + ".#":        "2",
				protectionsvc.ProtectionResourceSchemaEnabled:           "true",
				protectionsvc.ProtectionResourceSchemaWaitForLE:         "600",
			})
		},
	}
}

func ddosReadLeavesClientEmptyCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(ddosRemote(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read leaves the client attribute empty because the api sends a number into a string attribute",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.NotEqual(t, ddosClientIDStr, state.Attributes[protectionsvc.ProtectionResourceSchemaClient])
			require.Empty(t, state.Attributes[protectionsvc.ProtectionResourceSchemaClient])
		},
	}
}

func ddosReadEmptyGeoIPListCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	remote := ddosRemote()
	remote.GeoIPList = ""

	mc.Resources.On("Get", mock.Anything, resourceID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read turns an empty geoip list into a set holding one empty string",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionResourceSchemaGeoIPList + ".#": "1",
			})
			require.Contains(t, setValues(state, protectionsvc.ProtectionResourceSchemaGeoIPList), "")
		},
	}
}

func ddosReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).
		Return(nil, nil, fmt.Errorf("api error: resource doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "resource doesn't exist")
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func ddosReadNonNumericIDCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read rejects an id that is not a number",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    ddosName,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid syntax")
			fake.Resources.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
		},
	}
}

func ddosUpdateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.ResourceUpdateRequest

	updated := ddosRemote()
	updated.WAF = false

	mc.Resources.On("Get", mock.Anything, resourceID).Return(ddosRemote(), nil, nil).Once()
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.ResourceUpdateRequest) }).
		Return(updated, nil, nil)
	mc.Resources.On("Get", mock.Anything, resourceID).Return(updated, nil, nil)

	config := ddosConfig()
	config[protectionsvc.ProtectionResourceSchemaWAF] = false

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "update re-reads the resource first and carries the current ssl type over",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.NotNil(t, sent.SSLType)
			require.Equal(t, "custom", *sent.SSLType)
			require.False(t, sent.WAF)

			support.RequireStateID(t, state, resourceIDStr)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionResourceSchemaWAF: "false",
			})
		},
	}
}

func ddosUpdateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Get", mock.Anything, resourceID).Return(ddosRemote(), nil, nil)
	mc.Resources.On("Update", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: tls list is not allowed"))

	config := ddosConfig()
	config[protectionsvc.ProtectionResourceSchemaTLS] = []interface{}{"1"}

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "update surfaces the api error",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "tls list is not allowed")
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func ddosDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Delete", mock.Anything, resourceID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Resources.AssertCalled(t, "Delete", mock.Anything, resourceID)
		},
	}
}

func ddosDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Resources.On("Delete", mock.Anything, resourceID).
		Return(nil, fmt.Errorf("api error: resource is locked"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: ddosConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "resource is locked")
			support.RequireStateID(t, state, resourceIDStr)
		},
	}
}

func TestIntegrationProtectionResource_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionResourceResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		ddosCreateCase(),
		ddosCreateUnsetOptionsCase(),
		ddosCreateAPIFailureCase(),
		ddosCreateReadFailureCase(),
		ddosReadCase(),
		ddosReadLeavesClientEmptyCase(),
		ddosReadEmptyGeoIPListCase(),
		ddosReadAPIFailureCase(),
		ddosReadNonNumericIDCase(),
		ddosUpdateCase(),
		ddosUpdateAPIFailureCase(),
		ddosDeleteCase(),
		ddosDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
