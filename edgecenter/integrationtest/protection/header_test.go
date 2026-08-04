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
	headerKey      = "X-My-Header"
	headerValue    = "one"
	headerNewKey   = "X-Other-Header"
	headerNewValue = "two"
)

func headerConfig(key, value string) map[string]interface{} {
	return map[string]interface{}{
		protectionsvc.ProtectionHeaderSchemaResource: resourceIDStr,
		protectionsvc.ProtectionHeaderSchemaKey:      key,
		protectionsvc.ProtectionHeaderSchemaValue:    value,
	}
}

func headerRemote(key, value string) *protectionSDK.Header {
	return &protectionSDK.Header{ID: childID, Key: key, Value: value}
}

func headerCreateCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.HeaderCreateRequest

	mc.Headers.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.HeaderCreateRequest) }).
		Return(headerRemote(headerKey, headerValue), nil, nil)
	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(headerRemote(headerKey, headerValue), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create sends the key and the value and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, headerKey, sent.Key)
			require.Equal(t, headerValue, sent.Value)

			support.RequireStateID(t, state, compositeID)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionHeaderSchemaResource: resourceIDStr,
				protectionsvc.ProtectionHeaderSchemaKey:      headerKey,
				protectionsvc.ProtectionHeaderSchemaValue:    headerValue,
			})
		},
	}
}

func headerCreateAcceptsAnyKeyCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.HeaderCreateRequest

	weird := "not a valid: header name"

	mc.Headers.On("Create", mock.Anything, resourceID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(2).(*protectionSDK.HeaderCreateRequest) }).
		Return(headerRemote(weird, headerValue), nil, nil)
	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(headerRemote(weird, headerValue), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create forwards a header name the http grammar forbids because the attribute has no validator",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: headerConfig(weird, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, weird, sent.Key)
		},
	}
}

func headerCreateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: header already exists"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "header already exists")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Headers.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func headerCreateReadFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Create", mock.Anything, resourceID, mock.Anything).
		Return(headerRemote(headerKey, headerValue), nil, nil)
	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:      "create reports success and keeps the composite id when the follow-up read fails",
		Op:        support.OpApply,
		Prepare:   func() *protectionmock.MockedProtection { return mc },
		NewConfig: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func headerReadCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(headerRemote(headerNewKey, headerNewValue), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read replaces both the key and the value with what the api reports",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionHeaderSchemaResource: resourceIDStr,
				protectionsvc.ProtectionHeaderSchemaKey:      headerNewKey,
				protectionsvc.ProtectionHeaderSchemaValue:    headerNewValue,
			})
		},
	}
}

func headerReadAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(nil, nil, fmt.Errorf("api error: header doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "read surfaces the api error and keeps the id in state",
		Op:           support.OpRead,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "header doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func headerUpdateKeyCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.HeaderCreateRequest

	mc.Headers.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.HeaderCreateRequest) }).
		Return(headerRemote(headerNewKey, headerValue), nil, nil)
	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(headerRemote(headerNewKey, headerValue), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "renaming the header updates it in place instead of replacing it",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		NewConfig:    headerConfig(headerNewKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent)
			require.Equal(t, headerNewKey, sent.Key)
			require.Equal(t, headerValue, sent.Value)

			support.RequireStateID(t, state, compositeID)
			fake.Headers.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func headerUpdateValueCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	var sent *protectionSDK.HeaderCreateRequest

	mc.Headers.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Run(func(args mock.Arguments) { sent = args.Get(3).(*protectionSDK.HeaderCreateRequest) }).
		Return(headerRemote(headerKey, headerNewValue), nil, nil)
	mc.Headers.On("Get", mock.Anything, resourceID, childID).
		Return(headerRemote(headerKey, headerNewValue), nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "changing the value sends the unchanged key along with it",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		NewConfig:    headerConfig(headerKey, headerNewValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, headerKey, sent.Key)
			require.Equal(t, headerNewValue, sent.Value)
			support.RequireStateAttrs(t, state, map[string]string{
				protectionsvc.ProtectionHeaderSchemaValue: headerNewValue,
			})
		},
	}
}

func headerUpdateAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Update", mock.Anything, resourceID, childID, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: header name is not allowed"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "update surfaces the api error",
		Op:           support.OpApply,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		NewConfig:    headerConfig(headerNewKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "header name is not allowed")
			support.RequireStateID(t, state, compositeID)
			fake.Headers.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func headerDeleteCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Delete", mock.Anything, resourceID, childID).Return(nil, nil)

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete calls the api and clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *protectionmock.MockedProtection) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state)
			fake.Headers.AssertCalled(t, "Delete", mock.Anything, resourceID, childID)
		},
	}
}

func headerDeleteAPIFailureCase() support.ResourceCase[*protectionmock.MockedProtection] {
	mc := protectionmock.NewMockedProtection()

	mc.Headers.On("Delete", mock.Anything, resourceID, childID).
		Return(nil, fmt.Errorf("api error: header doesn't exist"))

	return support.ResourceCase[*protectionmock.MockedProtection]{
		Name:         "delete surfaces the api error and keeps the id in state",
		Op:           support.OpDelete,
		Prepare:      func() *protectionmock.MockedProtection { return mc },
		CurrentID:    compositeID,
		CurrentState: headerConfig(headerKey, headerValue),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *protectionmock.MockedProtection) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "header doesn't exist")
			support.RequireStateID(t, state, compositeID)
		},
	}
}

func TestIntegrationProtectionHeader_TableDriven(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionHeaderResource)

	cases := []support.ResourceCase[*protectionmock.MockedProtection]{
		headerCreateCase(),
		headerCreateAcceptsAnyKeyCase(),
		headerCreateAPIFailureCase(),
		headerCreateReadFailureCase(),
		headerReadCase(),
		headerReadAPIFailureCase(),
		headerUpdateKeyCase(),
		headerUpdateValueCase(),
		headerUpdateAPIFailureCase(),
		headerDeleteCase(),
		headerDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*protectionmock.MockedProtection])
}
