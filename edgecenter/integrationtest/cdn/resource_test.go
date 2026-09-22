//go:build integration

package cdn_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/origingroups"
	"github.com/Edge-Center/edgecentercdn-go/resources"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testCDNResourceID       = 1001
	testCDNResourceCname    = "cdn.example.com"
	testCDNResourceOrigin   = "1.2.3.4:8080"
	testCDNResourceOriginGr = 55
	testCDNResourceCache    = "1800s"
)

func cdnResourceConfig(description string) map[string]interface{} {
	return map[string]interface{}{
		"cname":               testCDNResourceCname,
		"description":         description,
		"origin":              testCDNResourceOrigin,
		"origin_protocol":     "HTTPS",
		"secondary_hostnames": []interface{}{"a.example.com", "b.example.com"},
		"options": []interface{}{
			map[string]interface{}{
				"browser_cache_settings": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"value":   testCDNResourceCache,
					},
				},
			},
		},
	}
}

func sampleCDNResource(description string) *resources.Resource {
	return &resources.Resource{
		ID:                 testCDNResourceID,
		Cname:              testCDNResourceCname,
		Description:        description,
		OriginGroup:        testCDNResourceOriginGr,
		OriginProtocol:     resources.HTTPSProtocol,
		SecondaryHostnames: []string{"a.example.com", "b.example.com"},
		Status:             "active",
		Active:             true,
		Options: &cdnsdk.ResourceOptions{
			LocationOptions: cdnsdk.LocationOptions{
				BrowserCacheSettings: &cdnsdk.BrowserCacheSettings{
					Enabled: true,
					Value:   testCDNResourceCache,
				},
			},
		},
	}
}

func implicitOriginGroupName() string {
	return fmt.Sprintf("Origins for %s (%d)", testCDNResourceCname, testCDNResourceID)
}

func sampleImplicitOriginGroup(id int64, source string) *origingroups.OriginGroup {
	return &origingroups.OriginGroup{
		ID:      id,
		Name:    implicitOriginGroupName(),
		Origins: []origingroups.Origin{{ID: 1, Source: source, Enabled: true}},
	}
}

func mockImplicitOriginGroup(mc *cdnmock.MockedCDN, source string) {
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).
		Return(sampleImplicitOriginGroup(testCDNResourceOriginGr, source), nil)
}

func hasSingleOrigin(origins []origingroups.OriginRequest, source string) bool {
	return len(origins) == 1 && origins[0].Source == source && origins[0].Enabled && !origins[0].Backup
}

func hasBothHostnames(hostnames []string) bool {
	if len(hostnames) != 2 {
		return false
	}

	seen := map[string]bool{}
	for _, h := range hostnames {
		seen[h] = true
	}

	return seen["a.example.com"] && seen["b.example.com"]
}

func cdnResourceCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Create", mock.Anything,
		mock.MatchedBy(func(req *resources.CreateRequest) bool {
			return req.Cname == testCDNResourceCname &&
				req.Description == "tf test" &&
				req.Origin == testCDNResourceOrigin &&
				req.OriginProtocol == resources.HTTPSProtocol &&
				hasBothHostnames(req.SecondaryHostnames) &&
				req.Options != nil &&
				req.Options.BrowserCacheSettings != nil &&
				req.Options.BrowserCacheSettings.Value == testCDNResourceCache
		}),
	).Return(sampleCDNResource("tf test"), nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(sampleCDNResource("tf test"), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCDNResourceID))
			support.RequireStateAttrs(t, state, map[string]string{
				"cname":                 testCDNResourceCname,
				"description":           "tf test",
				"origin_protocol":       "HTTPS",
				"origin_group":          fmt.Sprintf("%d", testCDNResourceOriginGr),
				"status":                "active",
				"active":                "true",
				"secondary_hostnames.#": "2",
				"options.0.browser_cache_settings.0.enabled": "true",
				"options.0.browser_cache_settings.0.value":   testCDNResourceCache,
			})
		},
	}
}

func cdnResourceCreateDedupesHostnamesCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Create", mock.Anything,
		mock.MatchedBy(func(req *resources.CreateRequest) bool {
			return len(req.SecondaryHostnames) == 1 && req.SecondaryHostnames[0] == "a.example.com"
		}),
	).Return(sampleCDNResource("tf test"), nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(sampleCDNResource("tf test"), nil)

	config := cdnResourceConfig("tf test")
	config["secondary_hostnames"] = []interface{}{"a.example.com", "a.example.com"}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create deduplicates secondary hostnames",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCDNResourceID))
		},
	}
}

func cdnResourceReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	drifted := sampleCDNResource("changed-out-of-band")
	drifted.Status = "suspended"
	drifted.Active = false
	drifted.OriginProtocol = resources.HTTPProtocol
	drifted.Options.BrowserCacheSettings.Value = "60s"

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(drifted, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read overwrites state with API values",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCDNResourceID))
			support.RequireStateAttrs(t, state, map[string]string{
				"description":     "changed-out-of-band",
				"origin_group":    fmt.Sprintf("%d", testCDNResourceOriginGr),
				"origin_protocol": "HTTP",
				"status":          "suspended",
				"active":          "false",
				"options.0.browser_cache_settings.0.value": "60s",
			})
		},
	}
}

func cdnResourceUpdateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const newDescription = "tf test updated"

	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.Description == newDescription &&
				req.OriginProtocol == resources.HTTPSProtocol &&
				hasBothHostnames(req.SecondaryHostnames)
		}),
	).Return(sampleCDNResource(newDescription), nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(sampleCDNResource(newDescription), nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update description",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		NewConfig:    cdnResourceConfig(newDescription),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"description": newDescription,
			})
		},
	}
}

func cdnResourceDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Delete", mock.Anything, int64(testCDNResourceID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete resource",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func cdnResourceCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: cname already exists"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "cname already exists")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func cdnResourceDeleteAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Delete", mock.Anything, int64(testCDNResourceID)).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCDNResourceID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func cdnResourceCreateWithOriginGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Create", mock.Anything,
		mock.MatchedBy(func(req *resources.CreateRequest) bool {
			return req.OriginGroup == testCDNResourceOriginGr && req.Origin == ""
		}),
	).Return(sampleCDNResource("tf test"), nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(sampleCDNResource("tf test"), nil)

	config := cdnResourceConfig("tf test")
	delete(config, "origin")
	config["origin_group"] = testCDNResourceOriginGr

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with origin_group instead of origin",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCDNResourceID))
		},
	}
}

func cdnResourceCreateWithSSLCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const certID = 3001

	withSSL := sampleCDNResource("tf test")
	withSSL.SSLEnabled = true
	withSSL.SSLData = certID

	mc.Resources.On("Create", mock.Anything,
		mock.MatchedBy(func(req *resources.CreateRequest) bool {
			return req.SSLEnabled && req.SSLData == certID
		}),
	).Return(withSSL, nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(withSSL, nil)

	config := cdnResourceConfig("tf test")
	config["ssl_enabled"] = true
	config["ssl_data"] = certID

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "create with an ssl certificate attached",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"ssl_enabled": "true",
				"ssl_data":    fmt.Sprintf("%d", certID),
			})
		},
	}
}

func cdnResourceUpdateSendsSSLDataPointerCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const certID = 3001

	withSSL := sampleCDNResource("tf test")
	withSSL.SSLEnabled = true
	withSSL.SSLData = certID

	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.SSLEnabled && req.SSLData != nil && *req.SSLData == certID
		}),
	).Return(withSSL, nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(withSSL, nil)

	current := cdnResourceConfig("tf test")
	updated := cdnResourceConfig("tf test")
	updated["ssl_enabled"] = true
	updated["ssl_data"] = certID

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update attaches an ssl certificate",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		NewConfig:    updated,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"ssl_enabled": "true",
				"ssl_data":    fmt.Sprintf("%d", certID),
			})
		},
	}
}

func cdnResourceUpdateOptionsCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	updated := sampleCDNResource("tf test")
	updated.Options.BrowserCacheSettings.Value = "900s"

	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.Options != nil &&
				req.Options.BrowserCacheSettings != nil &&
				req.Options.BrowserCacheSettings.Value == "900s"
		}),
	).Return(updated, nil)

	mockImplicitOriginGroup(mc, testCDNResourceOrigin)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(updated, nil)

	newConfig := cdnResourceConfig("tf test")
	newConfig["options"] = []interface{}{
		map[string]interface{}{
			"browser_cache_settings": []interface{}{
				map[string]interface{}{"enabled": true, "value": "900s"},
			},
		},
	}

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update sends changed options",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		NewConfig:    newConfig,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"options.0.browser_cache_settings.0.value": "900s",
			})
		},
	}
}

func cdnResourceUpdateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID), mock.Anything).
		Return(nil, fmt.Errorf("api error: origin group does not exist"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on update",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		NewConfig:    cdnResourceConfig("tf test updated"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin group does not exist")
			require.NotNil(t, state, "state must survive a failed update")
			require.Equal(t, fmt.Sprintf("%d", testCDNResourceID), state.ID)
		},
	}
}

func cdnResourceReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func cdnResourceReadInvalidIDCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read rejects a non numeric id",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    "not-a-number",
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
		},
	}
}

func cdnResourceUpdateOriginCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const newOrigin = "9.9.9.9:8080"

	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).
		Return(sampleImplicitOriginGroup(testCDNResourceOriginGr, testCDNResourceOrigin), nil).Once()
	mc.OriginGroups.On("Update", mock.Anything, int64(testCDNResourceOriginGr),
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return req.Name == implicitOriginGroupName() && hasSingleOrigin(req.Origins, newOrigin)
		}),
	).Return(sampleImplicitOriginGroup(testCDNResourceOriginGr, newOrigin), nil)
	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.OriginGroup == testCDNResourceOriginGr
		}),
	).Return(sampleCDNResource("tf test"), nil)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).
		Return(sampleImplicitOriginGroup(testCDNResourceOriginGr, newOrigin), nil)

	current := cdnResourceConfig("tf test")
	current["origin_group"] = testCDNResourceOriginGr
	updated := cdnResourceConfig("tf test")
	updated["origin"] = newOrigin

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update origin rewrites the implicit origin group",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		NewConfig:    updated,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			mc.OriginGroups.AssertCalled(t, "Update", mock.Anything, int64(testCDNResourceOriginGr), mock.Anything)
			mc.OriginGroups.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin":       newOrigin,
				"origin_group": fmt.Sprintf("%d", testCDNResourceOriginGr),
			})
		},
	}
}

func cdnResourceUpdateOriginOnSharedGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	const (
		newOrigin  = "9.9.9.9:8080"
		newGroupID = 77
	)

	shared := &origingroups.OriginGroup{
		ID:   testCDNResourceOriginGr,
		Name: "shared group",
		Origins: []origingroups.Origin{
			{ID: 1, Source: "a.origin.example", Enabled: true},
			{ID: 2, Source: "b.origin.example", Enabled: true},
		},
	}
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).Return(shared, nil)
	mc.OriginGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *origingroups.GroupRequest) bool {
			return strings.HasPrefix(req.Name, implicitOriginGroupName()+" (") &&
				len(req.Name) == len(implicitOriginGroupName())+19 &&
				hasSingleOrigin(req.Origins, newOrigin)
		}),
	).Return(sampleImplicitOriginGroup(newGroupID, newOrigin), nil)

	moved := sampleCDNResource("tf test")
	moved.OriginGroup = newGroupID
	mc.Resources.On("Update", mock.Anything, int64(testCDNResourceID),
		mock.MatchedBy(func(req *resources.UpdateRequest) bool {
			return req.OriginGroup == newGroupID
		}),
	).Return(moved, nil)
	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(moved, nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(newGroupID)).
		Return(sampleImplicitOriginGroup(newGroupID, newOrigin), nil)

	current := cdnResourceConfig("tf test")
	delete(current, "origin")
	current["origin_group"] = testCDNResourceOriginGr
	updated := cdnResourceConfig("tf test")
	updated["origin"] = newOrigin

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update origin on a shared group creates a private group",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		NewConfig:    updated,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			mc.OriginGroups.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin":       newOrigin,
				"origin_group": fmt.Sprintf("%d", newGroupID),
			})
		},
	}
}

func cdnResourceUpdateOriginGroupAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).
		Return(sampleImplicitOriginGroup(testCDNResourceOriginGr, testCDNResourceOrigin), nil)
	mc.OriginGroups.On("Update", mock.Anything, int64(testCDNResourceOriginGr), mock.Anything).
		Return(nil, fmt.Errorf("api error: origin does not resolve"))

	current := cdnResourceConfig("tf test")
	current["origin_group"] = testCDNResourceOriginGr
	updated := cdnResourceConfig("tf test")
	updated["origin"] = "9.9.9.9:8080"

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on origin group update stops the resource update",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		NewConfig:    updated,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "origin does not resolve")
			mc.Resources.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
			require.NotNil(t, state, "state must survive a failed update")
		},
	}
}

func cdnResourceReadFillsOriginCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mockImplicitOriginGroup(mc, testCDNResourceOrigin)

	current := cdnResourceConfig("tf test")
	delete(current, "origin")

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read fills origin from the implicit origin group",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin":       testCDNResourceOrigin,
				"origin_group": fmt.Sprintf("%d", testCDNResourceOriginGr),
			})
		},
	}
}

func cdnResourceReadClearsOriginForSharedGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).Return(&origingroups.OriginGroup{
		ID:      testCDNResourceOriginGr,
		Name:    "shared group",
		Origins: []origingroups.Origin{{ID: 1, Source: "a.origin.example", Enabled: true}},
	}, nil)

	current := cdnResourceConfig("tf test")
	current["origin_group"] = testCDNResourceOriginGr

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read clears a stale origin for a shared group",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"origin":       "",
				"origin_group": fmt.Sprintf("%d", testCDNResourceOriginGr),
			})
		},
	}
}

func cdnResourceReadOriginGroupAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on origin group read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func cdnResourceReadClearsOriginForMultiOriginGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).Return(&origingroups.OriginGroup{
		ID:   testCDNResourceOriginGr,
		Name: implicitOriginGroupName(),
		Origins: []origingroups.Origin{
			{ID: 1, Source: testCDNResourceOrigin, Enabled: true},
			{ID: 2, Source: "backup.origin.example", Enabled: true, Backup: true},
		},
	}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read clears origin when the implicit group grew a second origin",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: cdnResourceConfig("tf test"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"origin": ""})
		},
	}
}

func cdnResourceReadMatchesImplicitGroupCaseInsensitivelyCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Resources.On("Get", mock.Anything, int64(testCDNResourceID)).Return(sampleCDNResource("tf test"), nil)
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).Return(&origingroups.OriginGroup{
		ID:      testCDNResourceOriginGr,
		Name:    strings.ToUpper(implicitOriginGroupName()) + " (deadbeef)",
		Origins: []origingroups.Origin{{ID: 1, Source: testCDNResourceOrigin, Enabled: true}},
	}, nil)

	current := cdnResourceConfig("tf test")
	delete(current, "origin")

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read recognises a suffixed implicit group regardless of case",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"origin": testCDNResourceOrigin})
		},
	}
}

func cdnResourceUpdateOriginRefusesAuthorizedGroupCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	authorized := sampleImplicitOriginGroup(testCDNResourceOriginGr, testCDNResourceOrigin)
	authorized.Authorization = &origingroups.Authorization{AuthType: "aws_signature_v4", AccessKeyID: "key"}
	mc.OriginGroups.On("Get", mock.Anything, int64(testCDNResourceOriginGr)).Return(authorized, nil)

	current := cdnResourceConfig("tf test")
	current["origin_group"] = testCDNResourceOriginGr
	updated := cdnResourceConfig("tf test")
	updated["origin"] = "9.9.9.9:8080"

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "update origin refuses to touch an implicit group with authorization",
		Op:           support.OpApply,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCDNResourceID),
		CurrentState: current,
		NewConfig:    updated,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "has authorization configured")
			mc.OriginGroups.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
			mc.OriginGroups.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			mc.Resources.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func TestIntegrationCDNResource_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_resource")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		cdnResourceCreateCase(),
		cdnResourceCreateWithOriginGroupCase(),
		cdnResourceCreateWithSSLCase(),
		cdnResourceCreateDedupesHostnamesCase(),
		cdnResourceReadCase(),
		cdnResourceUpdateCase(),
		cdnResourceUpdateOptionsCase(),
		cdnResourceUpdateSendsSSLDataPointerCase(),
		cdnResourceUpdateOriginCase(),
		cdnResourceUpdateOriginOnSharedGroupCase(),
		cdnResourceUpdateOriginGroupAPIFailureCase(),
		cdnResourceReadFillsOriginCase(),
		cdnResourceReadClearsOriginForSharedGroupCase(),
		cdnResourceReadClearsOriginForMultiOriginGroupCase(),
		cdnResourceReadMatchesImplicitGroupCaseInsensitivelyCase(),
		cdnResourceUpdateOriginRefusesAuthorizedGroupCase(),
		cdnResourceReadOriginGroupAPIFailureCase(),
		cdnResourceDeleteCase(),
		cdnResourceCreateAPIFailureCase(),
		cdnResourceUpdateAPIFailureCase(),
		cdnResourceReadAPIFailureCase(),
		cdnResourceReadInvalidIDCase(),
		cdnResourceDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
