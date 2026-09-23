//go:build integration

package cdn_test

import (
	"fmt"
	"math"
	"strconv"
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

func pointsTo[T comparable](got *T, want T) bool {
	return got != nil && *got == want
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
				pointsTo(req.Active, true) &&
				pointsTo(req.Weight, 10) &&
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

func ruleCreateDisabledWithLowestWeightCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	config := ruleConfig(testRuleName)
	config["active"] = false
	config["weight"] = 1

	created := sampleRule(testRuleName)
	created.Active = false
	created.Weight = 1

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
		mock.MatchedBy(func(req *rules.CreateRequest) bool {
			return pointsTo(req.Active, false) && pointsTo(req.Weight, 1)
		}),
	).Return(created, nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(created, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create sends active false and weight 1",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"active": "false",
				"weight": "1",
			})
		},
	}
}

func ruleCreateWithoutActiveAndWeightCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	config := ruleConfig(testRuleName)
	delete(config, "active")
	delete(config, "weight")

	mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
		mock.MatchedBy(func(req *rules.CreateRequest) bool {
			return req.Active == nil && req.Weight == nil
		}),
	).Return(sampleRule(testRuleName), nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(sampleRule(testRuleName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create leaves unset active and weight to API defaults",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"active": "true",
				"weight": "10",
			})
		},
	}
}

func ruleUpdateDisablesCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	config := ruleConfig(testRuleName)
	config["active"] = false

	disabled := sampleRule(testRuleName)
	disabled.Active = false

	mc.Rules.On("Update", mock.Anything, int64(testRuleResourceID), int64(testRuleID),
		mock.MatchedBy(func(req *rules.UpdateRequest) bool {
			return pointsTo(req.Active, false) && pointsTo(req.Weight, 10)
		}),
	).Return(disabled, nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(disabled, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update sends active false",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"active": "false"})
		},
	}
}

func ruleUpdateWithoutActiveAndWeightCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const newName = "tf-rule-renamed"

	config := ruleConfig(newName)
	delete(config, "active")
	delete(config, "weight")

	mc.Rules.On("Update", mock.Anything, int64(testRuleResourceID), int64(testRuleID),
		mock.MatchedBy(func(req *rules.UpdateRequest) bool {
			return req.Name == newName && req.Active == nil && req.Weight == nil
		}),
	).Return(sampleRule(newName), nil)

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(sampleRule(newName), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update omits active and weight that are not in config",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testRuleID),
		CurrentState: ruleConfig(testRuleName),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   newName,
				"active": "true",
				"weight": "10",
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
			return req.Name == newName && req.Rule == testRulePattern && pointsTo(req.Active, true)
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
		ruleCreateDisabledWithLowestWeightCase(),
		ruleCreateWithoutActiveAndWeightCase(),
		ruleReadCase(),
		ruleUpdateCase(),
		ruleUpdateDisablesCase(),
		ruleUpdateWithoutActiveAndWeightCase(),
		ruleUpdateOptionsCase(),
		ruleDeleteCase(),
		ruleCreateAPIFailureCase(),
		ruleUpdateAPIFailureCase(),
		ruleReadAPIFailureCase(),
		ruleDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCaseWithRawConfig[*cdnmock.MockedCDN])
}

func TestIntegrationRule_WeightValidation(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_rule")

	tests := []struct {
		weight int64
		valid  bool
	}{
		{weight: -1, valid: false},
		{weight: 0, valid: false},
		{weight: 1, valid: true},
		{weight: math.MaxInt32, valid: true},
		{weight: math.MaxInt32 + 1, valid: false},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatInt(tt.weight, 10), func(t *testing.T) {
			config := ruleConfig(testRuleName)
			config["weight"] = float64(tt.weight)

			diags := resource.Validate(terraform.NewResourceConfigRaw(config))
			require.Equal(t, tt.valid, !diags.HasError(), "%v", diags)
		})
	}
}
