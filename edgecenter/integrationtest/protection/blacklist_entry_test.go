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
	blacklistIP    = "198.51.100.7"
	blacklistNewIP = "198.51.100.8"
	blacklistCIDR  = "198.51.100.0/27"
)

func blacklistConfig(ip string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionBlacklistEntrySchemaResource: resourceIDStr,
		protectionsvc.ProtectionBlacklistEntrySchemaIP:       ip,
	}
}

func blacklistRemote(ip string) *protectionSDK.Blacklist {
	return &protectionSDK.Blacklist{ID: childID, IP: ip}
}

func blacklistCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.BlacklistCreateRequest

	mc.Blacklists.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.BlacklistCreateRequest) }).
		Return(blacklistRemote(blacklistIP), nil, nil)
	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(blacklistRemote(blacklistIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends the ip and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, blacklistIP, sent.IP)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionBlacklistEntrySchemaResource: resourceIDStr,
				protectionsvc.ProtectionBlacklistEntrySchemaIP:       blacklistIP,
			})
		},
	}
}

func blacklistCreateCIDRCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.BlacklistCreateRequest

	mc.Blacklists.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.BlacklistCreateRequest) }).
		Return(blacklistRemote(blacklistCIDR), nil, nil)
	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(blacklistRemote(blacklistCIDR), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create passes a cidr through untouched because the attribute has no validator",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: blacklistConfig(blacklistCIDR),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, blacklistCIDR, sent.IP)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionBlacklistEntrySchemaIP: blacklistCIDR,
			})
		},
	}
}

func blacklistCreateNonNumericResourceCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	config := blacklistConfig(blacklistIP)
	config[protectionsvc.ProtectionBlacklistEntrySchemaResource] = "not-a-number"

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create rejects a resource reference that is not a number",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid syntax")
			fake.Blacklists.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func blacklistCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: ip is already blacklisted"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "ip is already blacklisted")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Blacklists.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func blacklistCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(blacklistRemote(blacklistIP), nil, nil)
	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the composite id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func blacklistReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(blacklistRemote(blacklistNewIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read replaces the ip in state with the one the api reports",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionBlacklistEntrySchemaResource: resourceIDStr,
				protectionsvc.ProtectionBlacklistEntrySchemaIP:       blacklistNewIP,
			})
		},
	}
}

func blacklistReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: entry doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entry doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func blacklistReadMalformedIDCase(name, id, wantErr string) support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         name,
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    id,
		CurrentState: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, wantErr)
			fake.Blacklists.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func blacklistUpdateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.BlacklistCreateRequest

	mc.Blacklists.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.BlacklistCreateRequest) }).
		Return(blacklistRemote(blacklistNewIP), nil, nil)
	mc.Blacklists.On("Get", mock.Anything, resourceID, childID).
		Return(blacklistRemote(blacklistNewIP), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "changing the ip updates the entry in place",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		NewConfig:    blacklistConfig(blacklistNewIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, blacklistNewIP, sent.IP)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionBlacklistEntrySchemaIP: blacklistNewIP,
			})
		},
	}
}

func blacklistUpdateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: ip is malformed"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "update surfaces the api error",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		NewConfig:    blacklistConfig(blacklistNewIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "ip is malformed")
			support.RequireStateID(t, state, compositeID)
			fake.Blacklists.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func blacklistDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Blacklists.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
		},
	}
}

func blacklistDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Blacklists.On("Delete", mock.Anything, resourceID, childID).
		Return(nil, fmt.Errorf("api error: entry doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: blacklistConfig(blacklistIP),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "entry doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionBlacklistEntry_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionBlacklistEntryResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		blacklistCreateCase(),
		blacklistCreateCIDRCase(),
		blacklistCreateNonNumericResourceCase(),
		blacklistCreateAPIFailureCase(),
		blacklistCreateReadFailureCase(),
		blacklistReadCase(),
		blacklistReadAPIFailureCase(),
		blacklistReadMalformedIDCase(
			"read rejects an id without a separator", resourceIDStr, "wrong input id"),
		blacklistReadMalformedIDCase(
			"read rejects an id with three parts", "42:7:9", "wrong input id"),
		blacklistReadMalformedIDCase(
			"read rejects an empty id", " ", "wrong input id"),
		blacklistReadMalformedIDCase(
			"read rejects a composite id whose parts are not numbers", "a:b", "invalid syntax"),
		blacklistUpdateCase(),
		blacklistUpdateAPIFailureCase(),
		blacklistDeleteCase(),
		blacklistDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
