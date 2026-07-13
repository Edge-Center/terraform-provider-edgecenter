//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/rules"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testRuleResourceID = 100
	testRuleID         = 900
	testRuleName       = "tf-rule"
	testRulePattern    = "/images/*"
	testRuleCacheValue = "3600s"
)

func ruleConfig(name string) map[string]interface{} {
	return map[string]interface{}{
		"resource_id":     testRuleResourceID,
		"name":            name,
		"rule":            testRulePattern,
		"active":          true,
		"weight":          10,
		"origin_protocol": "HTTPS",
		"options": []interface{}{
			map[string]interface{}{
				"browser_cache_settings": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"value":   testRuleCacheValue,
					},
				},
			},
		},
	}
}

func sampleRule(name string) *rules.Rule {
	return &rules.Rule{
		ID:             testRuleID,
		Name:           name,
		Pattern:        testRulePattern,
		Active:         true,
		Weight:         10,
		OriginProtocol: "HTTPS",
		Options: &cdnsdk.LocationOptions{
			BrowserCacheSettings: &cdnsdk.BrowserCacheSettings{
				Enabled: true,
				Value:   testRuleCacheValue,
			},
		},
	}
}

func ruleCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
		mock.MatchedBy(func(req *rules.CreateRequest) bool {
			return req.Name == testRuleName &&
				req.Rule == testRulePattern &&
				req.Active &&
				req.Weight == 10 &&
				req.OriginGroup == nil &&
				req.OverrideOriginProtocol != nil && *req.OverrideOriginProtocol == "HTTPS" &&
				req.Options != nil &&
				req.Options.BrowserCacheSettings != nil &&
				req.Options.BrowserCacheSettings.Enabled &&
				req.Options.BrowserCacheSettings.Value == testRuleCacheValue
		}),
	).Return(sampleRule(testRuleName), nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(sampleRule(testRuleName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testRuleID))
			support.RequireStateAttrs(t, state, map[string]string{
				"resource_id":     fmt.Sprintf("%d", testRuleResourceID),
				"name":            testRuleName,
				"rule":            testRulePattern,
				"active":          "true",
				"weight":          "10",
				"origin_protocol": "HTTPS",
				"options.0.browser_cache_settings.0.enabled": "true",
				"options.0.browser_cache_settings.0.value":   testRuleCacheValue,
			})
		},
	}
}

func ruleCreateWithOriginGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const originGroupID = 55

	rule := sampleRule(testRuleName)
	originGroup := originGroupID
	rule.OriginGroup = &originGroup

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
		mock.MatchedBy(func(req *rules.CreateRequest) bool {
			return req.OriginGroup != nil && *req.OriginGroup == originGroupID
		}),
	).Return(rule, nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).Return(rule, nil)

	config := ruleConfig(testRuleName)
	config["origin_group"] = originGroupID

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create passes origin_group when set",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin_group": fmt.Sprintf("%d", originGroupID),
			})
		},
	}
}

func ruleReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	drifted := sampleRule("renamed-out-of-band")
	drifted.Pattern = "/video/*"
	drifted.Weight = 42
	drifted.Active = false
	drifted.Options.BrowserCacheSettings.Value = "7200s"

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).Return(drifted, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read overwrites state with API values",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testRuleID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   "renamed-out-of-band",
				"rule":   "/video/*",
				"weight": "42",
				"active": "false",
				"options.0.browser_cache_settings.0.value": "7200s",
			})
		},
	}
}

func ruleUpdateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const newName = "tf-rule-renamed"

	mc.Rules.On("Update", mock.Anything, int64(testRuleResourceID), int64(testRuleID),
		mock.MatchedBy(func(req *rules.UpdateRequest) bool {
			return req.Name == newName && req.Rule == testRulePattern && req.Active
		}),
	).Return(sampleRule(newName), nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(sampleRule(newName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update rule name",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		NewConfig:    ruleConfig(newName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": newName,
			})
		},
	}
}

func ruleDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Delete", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete rule",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func ruleCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID), mock.Anything).
		Return(nil, fmt.Errorf("api error: invalid rule pattern"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid rule pattern")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func ruleDeleteAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Delete", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testRuleID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func ruleCreateWithoutOriginProtocolCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	inherited := sampleRule(testRuleName)

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
		mock.MatchedBy(func(req *rules.CreateRequest) bool {
			return req.OverrideOriginProtocol == nil
		}),
	).Return(inherited, nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).Return(inherited, nil)

	config := ruleConfig(testRuleName)
	delete(config, "origin_protocol")

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create without origin_protocol sends a null override",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testRuleID))
		},
	}
}

func ruleUpdateOptionsCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	updated := sampleRule(testRuleName)
	updated.Options.BrowserCacheSettings.Value = "120s"

	mc.Rules.On("Update", mock.Anything, int64(testRuleResourceID), int64(testRuleID),
		mock.MatchedBy(func(req *rules.UpdateRequest) bool {
			return req.Options != nil &&
				req.Options.BrowserCacheSettings != nil &&
				req.Options.BrowserCacheSettings.Value == "120s"
		}),
	).Return(updated, nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).Return(updated, nil)

	newConfig := ruleConfig(testRuleName)
	newConfig["options"] = []interface{}{
		map[string]interface{}{
			"browser_cache_settings": []interface{}{
				map[string]interface{}{"enabled": true, "value": "120s"},
			},
		},
	}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update sends changed options",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		NewConfig:    newConfig,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"options.0.browser_cache_settings.0.value": "120s",
			})
		},
	}
}

func ruleUpdateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Update", mock.Anything, int64(testRuleResourceID), int64(testRuleID), mock.Anything).
		Return(nil, fmt.Errorf("api error: invalid rule pattern"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on update",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		NewConfig:    ruleConfig("tf-rule-renamed"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid rule pattern")
			require.NotNil(t, state, "state must survive a failed update")
		},
	}
}

func ruleReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func TestIntegrationRule_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_rule")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		ruleCreateCase(),
		ruleCreateWithOriginGroupCase(),
		ruleCreateWithoutOriginProtocolCase(),
		ruleReadCase(),
		ruleUpdateCase(),
		ruleUpdateOptionsCase(),
		ruleDeleteCase(),
		ruleCreateAPIFailureCase(),
		ruleUpdateAPIFailureCase(),
		ruleReadAPIFailureCase(),
		ruleDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
