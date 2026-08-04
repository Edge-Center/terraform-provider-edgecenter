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
	whitelistIP    = "198.51.100.42"
	whitelistNewIP = "198.51.100.43"
)

func whitelistConfig(ip string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionWhitelistEntrySchemaResource: resourceIDStr,
		protectionsvc.ProtectionWhitelistEntrySchemaIP:       ip,
	}
}

func whitelistRemote(ip string) *protectionSDK.Whitelist {
	return &protectionSDK.Whitelist{ID: childID, IP: ip}
}

func whitelistCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.WhitelistCreateRequest

	mc.Whitelists.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.WhitelistCreateRequest) }).
		Return(whitelistRemote(whitelistIP), nil, nil)
	mc.Whitelists.On("Get", mock.Anything, resourceID, childID).
		Return(whitelistRemote(whitelistIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends the ip and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, whitelistIP, sent.IP)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionWhitelistEntrySchemaResource: resourceIDStr,
				protectionsvc.ProtectionWhitelistEntrySchemaIP:       whitelistIP,
			})
		},
	}
}

func whitelistCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: ip is already whitelisted"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "ip is already whitelisted")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Whitelists.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func whitelistCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(whitelistRemote(whitelistIP), nil, nil)
	mc.Whitelists.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the composite id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func whitelistReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Get", mock.Anything, resourceID, childID).
		Return(whitelistRemote(whitelistNewIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read replaces the ip in state with the one the api reports",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionWhitelistEntrySchemaResource: resourceIDStr,
				protectionsvc.ProtectionWhitelistEntrySchemaIP:       whitelistNewIP,
			})
		},
	}
}

func whitelistReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: entry doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entry doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func whitelistReadMalformedIDCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read rejects an id without a separator",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong input id")
			fake.Whitelists.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func whitelistUpdateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.WhitelistCreateRequest

	mc.Whitelists.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.WhitelistCreateRequest) }).
		Return(whitelistRemote(whitelistNewIP), nil, nil)
	mc.Whitelists.On("Get", mock.Anything, resourceID, childID).
		Return(whitelistRemote(whitelistNewIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "changing the ip updates the entry in place",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		NewConfig:    whitelistConfig(whitelistNewIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, whitelistNewIP, sent.IP)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionWhitelistEntrySchemaIP: whitelistNewIP,
			})
		},
	}
}

func whitelistReplaceOnResourceChangeCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	other := int64(43)

	mc.Whitelists.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)
	mc.Whitelists.On("Create", mock.Anything, other, mock.Anything).
		Return(whitelistRemote(whitelistIP), nil, nil)
	mc.Whitelists.On("Get", mock.Anything, other, childID).
		Return(whitelistRemote(whitelistIP), nil, nil)

	config := whitelistConfig(whitelistIP)
	config[protectionsvc.ProtectionWhitelistEntrySchemaResource] = "43"

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "moving the entry to another protection resource recreates it",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "43:7")
			fake.Whitelists.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
			fake.Whitelists.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func whitelistDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Whitelists.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
		},
	}
}

func whitelistDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Whitelists.On("Delete", mock.Anything, resourceID, childID).
		Return(nil, fmt.Errorf("api error: entry doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: whitelistConfig(whitelistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entry doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionWhitelistEntry_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionWhitelistEntryResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		whitelistCreateCase(),
		whitelistCreateAPIFailureCase(),
		whitelistCreateReadFailureCase(),
		whitelistReadCase(),
		whitelistReadAPIFailureCase(),
		whitelistReadMalformedIDCase(),
		whitelistUpdateCase(),
		whitelistReplaceOnResourceChangeCase(),
		whitelistDeleteCase(),
		whitelistDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
