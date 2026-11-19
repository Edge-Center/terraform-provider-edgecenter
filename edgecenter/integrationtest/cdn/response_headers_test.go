//go:build integration

package cdn_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/resources"
	"github.com/Edge-Center/edgecentercdn-go/rules"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

var apiAlwaysShownHeaders = []string{"server", "content-length", "connection", "content-type", "date"}

type headersPolicyCase struct {
	name       string
	mode       string
	configured []string
	stored     []string
}

func headersPolicyCases() []headersPolicyCase {
	return []headersPolicyCase{
		{
			name:       "hide with a mandatory header in mixed case",
			mode:       "hide",
			configured: []string{"Server"},
			stored:     apiAlwaysShownHeaders,
		},
		{
			name:       "hide with a custom header",
			mode:       "hide",
			configured: []string{"X-Custom-Header"},
			stored:     append(slices.Clone(apiAlwaysShownHeaders), "x-custom-header"),
		},
		{
			name:       "show with a custom header",
			mode:       "show",
			configured: []string{"X-Custom-Header", "Cache-Control"},
			stored:     []string{"x-custom-header", "cache-control"},
		},
	}
}

func headersPolicyOptions(mode string, excepted []string) []interface{} {
	values := make([]interface{}, 0, len(excepted))
	for _, header := range excepted {
		values = append(values, header)
	}

	return []interface{}{
		map[string]interface{}{
			"response_headers_hiding_policy": []interface{}{
				map[string]interface{}{
					"enabled":  true,
					"mode":     mode,
					"excepted": values,
				},
			},
		},
	}
}

func headersPolicy(mode string, excepted []string) *cdnsdk.ResponseHeadersHidingPolicy {
	return &cdnsdk.ResponseHeadersHidingPolicy{Enabled: true, Mode: mode, Excepted: slices.Clone(excepted)}
}

func sendsHeaders(policy *cdnsdk.ResponseHeadersHidingPolicy, mode string, want []string) bool {
	if policy == nil || policy.Mode != mode {
		return false
	}

	return slices.Equal(slices.Sorted(slices.Values(policy.Excepted)), slices.Sorted(slices.Values(want)))
}

func requireNoOptionsDiff(t *testing.T, diff *terraform.InstanceDiff) {
	t.Helper()

	if diff == nil {
		return
	}
	for key := range diff.Attributes {
		require.False(t, strings.HasPrefix(key, "options."), "unexpected diff on %s", key)
	}
}

func requireOptionsDiff(t *testing.T, diff *terraform.InstanceDiff) {
	t.Helper()

	require.NotNil(t, diff)
	for key := range diff.Attributes {
		if strings.HasPrefix(key, "options.") {
			return
		}
	}
	require.Fail(t, "expected a diff on options", "%v", diff.Attributes)
}

func headersPolicyResourceConfig(mode string, excepted []string) map[string]interface{} {
	return map[string]interface{}{
		"cname":   testCDNResourceCname,
		"origin":  testCDNResourceOrigin,
		"options": headersPolicyOptions(mode, excepted),
	}
}

func headersPolicyResource(mode string, excepted []string) *resources.Resource {
	return &resources.Resource{
		ID:          testCDNResourceID,
		Cname:       testCDNResourceCname,
		OriginGroup: testCDNResourceOriginGr,
		Status:      "active",
		Active:      true,
		Options: &cdnsdk.ResourceOptions{
			LocationOptions: cdnsdk.LocationOptions{ResponseHeadersHidingPolicy: headersPolicy(mode, excepted)},
		},
	}
}

func applyHeadersPolicyResource(t *testing.T, res *schema.Resource, tc headersPolicyCase) (*cdnmock.MockedCDN, *terraform.InstanceState) {
	t.Helper()

	mc := cdnmock.NewMockedCDN()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Create", mock.Anything,
		mock.MatchedBy(func(req *resources.CreateRequest) bool {
			return req.Options != nil && sendsHeaders(req.Options.ResponseHeadersHidingPolicy, tc.mode, tc.configured)
		}),
	).Return(headersPolicyResource(tc.mode, tc.stored), nil)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(headersPolicyResource(tc.mode, tc.stored), nil).Once()

	state, diags := support.ApplyConfig(
		t, context.Background(), res, nil, headersPolicyResourceConfig(tc.mode, tc.configured), mc.Config,
	)
	support.RequireNoErrorDiags(t, diags)
	require.NotNil(t, state)

	return mc, state
}

func refresh(t *testing.T, res *schema.Resource, state *terraform.InstanceState, meta interface{}) *terraform.InstanceState {
	t.Helper()

	data := support.NewResourceDataFromState(t, res, state)
	support.RequireNoErrorDiags(t, res.ReadContext(context.Background(), data, meta))

	refreshed := data.State()
	require.NotNil(t, refreshed)

	return refreshed
}

func TestIntegrationCDNResource_HeadersHidingPolicyPlansCleanAfterApply(t *testing.T) {
	t.Parallel()

	for _, tc := range headersPolicyCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := cdnResource(t, "edgecenter_cdn_resource")
			mc, state := applyHeadersPolicyResource(t, res, tc)

			mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
				Return(headersPolicyResource(tc.mode, tc.stored), nil).Once()
			state = refresh(t, res, state, mc.Config)

			diff, err := res.Diff(context.Background(), state,
				terraform.NewResourceConfigRaw(headersPolicyResourceConfig(tc.mode, tc.configured)), mc.Config)
			require.NoError(t, err)
			requireNoOptionsDiff(t, diff)
		})
	}
}

func TestIntegrationCDNResource_HeadersHidingPolicyDetectsDrift(t *testing.T) {
	t.Parallel()

	res := cdnResource(t, "edgecenter_cdn_resource")
	tc := headersPolicyCases()[1]
	mc, state := applyHeadersPolicyResource(t, res, tc)

	drifted := append(slices.Clone(apiAlwaysShownHeaders), "x-other-header")
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(headersPolicyResource(tc.mode, drifted), nil).Once()
	state = refresh(t, res, state, mc.Config)

	diff, err := res.Diff(context.Background(), state,
		terraform.NewResourceConfigRaw(headersPolicyResourceConfig(tc.mode, tc.configured)), mc.Config)
	require.NoError(t, err)
	requireOptionsDiff(t, diff)
}

func headersPolicyRuleConfig(mode string, excepted []string) map[string]interface{} {
	return map[string]interface{}{
		"resource_id": testRuleResourceID,
		"name":        testRuleName,
		"rule":        testRulePattern,
		"options":     headersPolicyOptions(mode, excepted),
	}
}

func headersPolicyRule(mode string, excepted []string) *rules.Rule {
	return &rules.Rule{
		ID:      testRuleID,
		Name:    testRuleName,
		Pattern: testRulePattern,
		Active:  true,
		Weight:  1,
		Options: &cdnsdk.LocationOptions{ResponseHeadersHidingPolicy: headersPolicy(mode, excepted)},
	}
}

func TestIntegrationRule_HeadersHidingPolicyPlansCleanAfterApply(t *testing.T) {
	t.Parallel()

	for _, tc := range headersPolicyCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := cdnResource(t, "edgecenter_cdn_rule")
			mc := cdnmock.NewMockedCDN()
			t.Cleanup(func() { mc.MockCleanup(t) })

			mc.Rules.On("Create", mock.Anything, int64(testRuleResourceID),
				mock.MatchedBy(func(req *rules.CreateRequest) bool {
					return req.Options != nil && sendsHeaders(req.Options.ResponseHeadersHidingPolicy, tc.mode, tc.configured)
				}),
			).Return(headersPolicyRule(tc.mode, tc.stored), nil)
			mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
				Return(headersPolicyRule(tc.mode, tc.stored), nil).Twice()

			config := headersPolicyRuleConfig(tc.mode, tc.configured)
			empty := &terraform.InstanceState{Attributes: map[string]string{}}
			empty.RawConfig = support.NewRawConfig(t, res, config)
			state, diags := support.ApplyConfig(t, context.Background(), res, empty, config, mc.Config)
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testRuleID))

			state = refresh(t, res, state, mc.Config)

			diff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), mc.Config)
			require.NoError(t, err)
			requireNoOptionsDiff(t, diff)
		})
	}
}
