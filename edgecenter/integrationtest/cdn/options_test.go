//go:build integration

package cdn_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/resources"
	"github.com/Edge-Center/edgecentercdn-go/rules"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

func cdnResourceNoOptionsConfig(description string) map[string]interface{} {
	return map[string]interface{}{
		"cname":       testCDNResourceCname,
		"origin":      testCDNResourceOrigin,
		"description": description,
	}
}

func cdnResourceReadNilOptionsCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(&resources.Resource{
			ID:     testCDNResourceID,
			Cname:  testCDNResourceCname,
			Status: "active",
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read survives a nil options object",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "1001",
		CurrentState: cdnResourceNoOptionsConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"status":    "active",
				"options.#": "0",
			})
		},
	}
}

// An API answer of `"options": {}` used to land in state as options.# = 1 with a nil
// element, which blew up the next apply. The state has to come from a real read.
func TestIntegrationCDNResource_UpdateAfterEmptyOptionsRead(t *testing.T) {
	t.Parallel()

	res := cdnResource(t, "edgecenter_cdn_resource")
	mc := cdnmock.NewMockedCDN()
	t.Cleanup(func() { mc.MockCleanup(t) })

	emptyOptions := func(description string) *resources.Resource {
		return &resources.Resource{
			ID:          testCDNResourceID,
			Cname:       testCDNResourceCname,
			Description: description,
			Status:      "active",
			Options:     &cdnsdk.ResourceOptions{},
		}
	}

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(emptyOptions("tf test"), nil).Once()
	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.Description == "changed" && req.Options == nil
		}),
	).Return(emptyOptions("changed"), nil)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(emptyOptions("changed"), nil).Once()

	seed := support.NewState(t, res, cdnResourceNoOptionsConfig("tf test"), "1001")
	data := support.NewResourceDataFromState(t, res, seed)

	diags := res.ReadContext(context.Background(), data, mc.Config)
	support.RequireNoErrorDiags(t, diags)

	afterRead := data.State()
	require.NotNil(t, afterRead)

	state, diags := support.ApplyConfig(
		t, context.Background(), res, afterRead, cdnResourceNoOptionsConfig("changed"), mc.Config,
	)

	support.RequireNoErrorDiags(t, diags)
	support.RequireStateAttrs(t, state, map[string]string{"description": "changed"})
}

func ruleReadNilOptionsCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Rules.On("Get", mock.Anything, int64(testRuleResourceID), int64(testRuleID)).
		Return(&rules.Rule{
			ID:      testRuleID,
			Name:    testRuleName,
			Pattern: testRulePattern,
		}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "read survives a nil options object",
		Op:        support.OpRead,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		CurrentID: "900",
		CurrentState: map[string]interface{}{
			"resource_id": testRuleResourceID,
			"name":        testRuleName,
			"rule":        testRulePattern,
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      testRuleName,
				"options.#": "0",
			})
		},
	}
}

func TestIntegrationCDNResource_NilOptions(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_resource")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		cdnResourceReadNilOptionsCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}

func TestIntegrationRule_NilOptions(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_rule")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		ruleReadNilOptionsCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
