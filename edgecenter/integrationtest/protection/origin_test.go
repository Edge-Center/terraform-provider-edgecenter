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

const (
	originIP      = "203.0.113.10"
	originNewIP   = "203.0.113.11"
	originComment = "primary backend"
)

func originConfig() map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionOriginSchemaResource:    resourceIDStr,
		protectionsvc.ProtectionOriginSchemaIP:          originIP,
		protectionsvc.ProtectionOriginSchemaMode:        "primary",
		protectionsvc.ProtectionOriginSchemaWeight:      100,
		protectionsvc.ProtectionOriginSchemaMaxFails:    2,
		protectionsvc.ProtectionOriginSchemaFailTimeout: 3,
		protectionsvc.ProtectionOriginSchemaComment:     originComment,
	}
}

func originRemote() *protectionSDK.Origin {
	return &protectionSDK.Origin{
		ID:          childID,
		IP:          originIP,
		Mode:        "primary",
		Weight:      100,
		MaxFails:    2,
		FailTimeout: 3,
		Comment:     originComment,
	}
}

func originCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.OriginCreateRequest

	mc.Origins.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.OriginCreateRequest) }).
		Return(originRemote(), nil, nil)
	mc.Origins.On("Get", mock.Anything, resourceID, childID).Return(originRemote(), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends every attribute and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, originIP, sent.IP)
			require.Equal(t, "primary", sent.Mode)
			require.Equal(t, 100, sent.Weight)
			require.Equal(t, 2, sent.MaxFails)
			require.Equal(t, 3, sent.FailTimeout)
			require.Equal(t, originComment, sent.Comment)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionOriginSchemaResource: resourceIDStr,
				protectionsvc.ProtectionOriginSchemaIP:       originIP,
				protectionsvc.ProtectionOriginSchemaWeight:   "100",
			})
		},
	}
}

func originCreateZeroTuningCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.OriginCreateRequest

	remote := originRemote()
	remote.Weight = 1
	remote.MaxFails = 1
	remote.FailTimeout = 10
	remote.Comment = ""

	mc.Origins.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.OriginCreateRequest) }).
		Return(remote, nil, nil)
	mc.Origins.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	config := originConfig()
	config[protectionsvc.ProtectionOriginSchemaWeight] = 0
	config[protectionsvc.ProtectionOriginSchemaMaxFails] = 0
	config[protectionsvc.ProtectionOriginSchemaFailTimeout] = 0
	config[protectionsvc.ProtectionOriginSchemaComment] = ""

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "zero tuning values never reach the wire and the api defaults land in state instead",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Zero(t, sent.Weight)
			require.Zero(t, sent.MaxFails)
			require.Zero(t, sent.FailTimeout)
			require.Empty(t, sent.Comment)

			body, err := json.Marshal(sent)
			require.NoError(t, err)
			for _, field := range []string{"origin_weight", "origin_max_fails", "origin_fail_timeout", "origin_comment"} {
				require.NotContainsf(t, string(body), field, "%s must be absent from the request body", field)
			}

			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionOriginSchemaWeight:      "1",
				protectionsvc.ProtectionOriginSchemaMaxFails:    "1",
				protectionsvc.ProtectionOriginSchemaFailTimeout: "10",
			})
		},
	}
}

func originCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: origin ip is already used"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin ip is already used")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Origins.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func originCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Create", mock.Anything, resourceID, mock.Anything).Return(originRemote(), nil, nil)
	mc.Origins.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the composite id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func originReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	remote := originRemote()
	remote.Mode = "backup"
	remote.Weight = 55

	mc.Origins.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read overwrites every attribute with what the api reports",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionOriginSchemaResource:    resourceIDStr,
				protectionsvc.ProtectionOriginSchemaIP:          originIP,
				protectionsvc.ProtectionOriginSchemaMode:        "backup",
				protectionsvc.ProtectionOriginSchemaWeight:      "55",
				protectionsvc.ProtectionOriginSchemaMaxFails:    "2",
				protectionsvc.ProtectionOriginSchemaFailTimeout: "3",
				protectionsvc.ProtectionOriginSchemaComment:     originComment,
			})
		},
	}
}

func originReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: origin doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func originReadMalformedIDCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read rejects an id without a separator",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    resourceIDStr,
		CurrentState: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong input id")
			fake.Origins.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func originUpdateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.OriginCreateRequest

	remote := originRemote()
	remote.Mode = "down"

	mc.Origins.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.OriginCreateRequest) }).
		Return(remote, nil, nil)
	mc.Origins.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	config := originConfig()
	config[protectionsvc.ProtectionOriginSchemaMode] = "down"

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "changing the mode updates the origin in place and resends every other attribute",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, "down", sent.Mode)
			require.Equal(t, originIP, sent.IP)
			require.Equal(t, 100, sent.Weight)

			support.RequireStateID(t, state, compositeID)
			fake.Origins.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func originReplaceOnIPChangeCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	remote := originRemote()
	remote.IP = originNewIP

	mc.Origins.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)
	mc.Origins.On("Create", mock.Anything, resourceID, mock.Anything).Return(remote, nil, nil)
	mc.Origins.On("Get", mock.Anything, resourceID, childID).Return(remote, nil, nil)

	config := originConfig()
	config[protectionsvc.ProtectionOriginSchemaIP] = originNewIP

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "changing the ip destroys the origin and creates a new one",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
			fake.Origins.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
			fake.Origins.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func originUpdateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: mode is not allowed"))

	config := originConfig()
	config[protectionsvc.ProtectionOriginSchemaMode] = "down"

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "update surfaces the api error",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "mode is not allowed")
			support.RequireStateID(t, state, compositeID)
			fake.Origins.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func originDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Origins.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
		},
	}
}

func originDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Origins.On("Delete", mock.Anything, resourceID, childID).
		Return(nil, fmt.Errorf("api error: origin doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: originConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionOrigin_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionOriginResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		originCreateCase(),
		originCreateZeroTuningCase(),
		originCreateAPIFailureCase(),
		originCreateReadFailureCase(),
		originReadCase(),
		originReadAPIFailureCase(),
		originReadMalformedIDCase(),
		originUpdateCase(),
		originReplaceOnIPChangeCase(),
		originUpdateAPIFailureCase(),
		originDeleteCase(),
		originDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
