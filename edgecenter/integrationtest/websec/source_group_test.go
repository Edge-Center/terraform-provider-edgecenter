//go:build integration

package websec_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sourcegroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
)

const testSourceGroupName = "origins"

func sourceGroupConfig(sources ...map[string]interface{}) map[string]interface{} {
	items := make([]interface{}, 0, len(sources))
	for _, source := range sources {
		items = append(items, source)
	}

	return map[string]interface{}{
		"name":   testSourceGroupName,
		"source": items,
	}
}

func minimalSourceGroupConfig() map[string]interface{} {
	return sourceGroupConfig(map[string]interface{}{"source": "203.0.113.10"})
}

func fullSourceGroupConfig() map[string]interface{} {
	config := sourceGroupConfig(
		map[string]interface{}{
			"source":       "203.0.113.10:8080",
			"weight":       60,
			"max_fails":    3,
			"fail_timeout": 10,
			"comment":      "main",
		},
		map[string]interface{}{
			"source": "backup.example.com",
			"mode":   "backup",
			"weight": 40,
		},
	)
	config["description"] = "main origins"
	config["use_next"] = false

	return config
}

func minimalSourceGroup() *sourcegroups.SourceGroup {
	return &sourcegroups.SourceGroup{
		ID:      testSourceGroupID,
		Name:    testSourceGroupName,
		UseNext: true,
		Sources: []sourcegroups.Source{
			{Source: "203.0.113.10", Mode: sourcegroups.SourceModePrimary, Weight: 50, MaxFails: 1, FailTimeout: 1},
		},
	}
}

func fullSourceGroup() *sourcegroups.SourceGroup {
	return &sourcegroups.SourceGroup{
		ID:          testSourceGroupID,
		Name:        testSourceGroupName,
		Description: "main origins",
		Sources: []sourcegroups.Source{
			{Source: "203.0.113.10:8080", Mode: sourcegroups.SourceModePrimary, Weight: 60, MaxFails: 3, FailTimeout: 10, Comment: "main"},
			{Source: "backup.example.com", Mode: sourcegroups.SourceModeBackup, Weight: 40, MaxFails: 1, FailTimeout: 1},
		},
	}
}

func fullSourceGroupState() map[string]string {
	return map[string]string{
		"name":                  testSourceGroupName,
		"description":           "main origins",
		"use_next":              "false",
		"source.#":              "2",
		"source.0.source":       "203.0.113.10:8080",
		"source.0.mode":         "primary",
		"source.0.weight":       "60",
		"source.0.max_fails":    "3",
		"source.0.fail_timeout": "10",
		"source.0.comment":      "main",
		"source.1.source":       "backup.example.com",
		"source.1.mode":         "backup",
		"source.1.weight":       "40",
		"source.1.max_fails":    "1",
		"source.1.fail_timeout": "1",
		"source.1.comment":      "",
	}
}

func TestIntegrationWebsecSourceGroup_Create(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	var sent *sourcegroups.GroupRequest

	nameTaken := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest,
		`{"errors": {"name": ["Source Group name should be unique."]}}`)
	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create sends every default explicitly and reads the group back",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Create", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(1).(*sourcegroups.GroupRequest) }).
					Return(minimalSourceGroup(), nil).Once()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(minimalSourceGroup(), nil).Once()

				return mc
			},
			NewConfig: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateID(t, state, testSourceGroupIDStr)

				require.Equal(t, &sourcegroups.GroupRequest{
					Name:    testSourceGroupName,
					UseNext: ptr(true),
					Sources: minimalSourceGroup().Sources,
				}, sent)

				support.RequireStateAttrs(t, state, map[string]string{
					"name":                  testSourceGroupName,
					"description":           "",
					"use_next":              "true",
					"source.#":              "1",
					"source.0.source":       "203.0.113.10",
					"source.0.mode":         "primary",
					"source.0.weight":       "50",
					"source.0.max_fails":    "1",
					"source.0.fail_timeout": "1",
					"source.0.comment":      "",
				})
			},
		},
		{
			Name: "a rejected create shows the api error and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Create", mock.Anything, mock.Anything).Return(nil, nameTaken).Once()

				return mc
			},
			NewConfig: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "name: Source Group name should be unique.")
				require.Nil(t, state)
			},
		},
		{
			Name: "create keeps the id when the read after it fails",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Create", mock.Anything, mock.Anything).Return(minimalSourceGroup(), nil).Once()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(nil, unavailable).Once()

				return mc
			},
			NewConfig: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testSourceGroupIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSourceGroup_Read(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "read maps the api answer into state and keeps the order of the sources",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(fullSourceGroup(), nil).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, fullSourceGroupState())
			},
		},
		{
			Name: "read removes a group the api cannot find from state",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read keeps the state when the api fails for another reason",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(nil, unavailable).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testSourceGroupIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSourceGroup_Update(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	var sent *sourcegroups.GroupRequest

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "update puts the whole group in the configured order and reads it back",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Update", mock.Anything, testSourceGroupID, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(2).(*sourcegroups.GroupRequest) }).
					Return(fullSourceGroup(), nil).Once()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(fullSourceGroup(), nil).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			NewConfig:    fullSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)

				require.Equal(t, &sourcegroups.GroupRequest{
					Name:        testSourceGroupName,
					UseNext:     ptr(false),
					Description: "main origins",
					Sources:     fullSourceGroup().Sources,
				}, sent)

				support.RequireStateAttrs(t, state, fullSourceGroupState())
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSourceGroup_Delete(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "delete removes a group nothing uses",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(minimalSourceGroup(), nil).Once()
				mc.SourceGroups.On("Delete", mock.Anything, testSourceGroupID).Return(nil).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete refuses a group that resources and locations use and lists them",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				group := minimalSourceGroup()
				group.RelatedResources = []sourcegroups.RelatedResource{
					{ID: 11, Domain: "example.com", Locations: []sourcegroups.RelatedLocation{{ID: 5, Name: "static"}, {ID: 6, Name: "api"}}},
					{ID: 12, Domain: "other.example.com", Locations: []sourcegroups.RelatedLocation{}},
				}

				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(group, nil).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *websecmock.MockedWebsec) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags,
					"source group 7 is still used by resource 11 (example.com) with locations 5 (static), 6 (api); resource 12 (other.example.com):")
				support.RequireStateID(t, state, testSourceGroupIDStr)
				mc.SourceGroups.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "delete of a group that is already gone succeeds without a delete call",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
				mc.SourceGroups.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "delete treats a 404 from the delete call as success",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SourceGroups.On("Get", mock.Anything, testSourceGroupID).Return(minimalSourceGroup(), nil).Once()
				mc.SourceGroups.On("Delete", mock.Anything, testSourceGroupID).Return(notFound()).Once()

				return mc
			},
			CurrentID:    testSourceGroupIDStr,
			CurrentState: minimalSourceGroupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSourceGroup_Validate(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	t.Run("a host name or an ipv4 address with an optional port is accepted", func(t *testing.T) {
		t.Parallel()

		for _, source := range []string{
			"203.0.113.10",
			"203.0.113.10:8080",
			"origin.example.com",
			"Origin.Example.COM:65535",
			"example.com.",
			"xn--80aswg.xn--p1ai",
		} {
			diags := resource.Validate(terraform.NewResourceConfigRaw(sourceGroupConfig(map[string]interface{}{"source": source})))
			require.Emptyf(t, diags, "source %q", source)
		}
	})

	t.Run("a scheme, a path, an ipv6 address or a bad port is rejected", func(t *testing.T) {
		t.Parallel()

		for _, source := range []string{
			"https://origin.example.com",
			"origin.example.com/app",
			"2001:db8::1",
			"[2001:db8::1]:443",
			"origin.example.com:0",
			"origin.example.com:080",
			"origin.example.com:65536",
			"origin.example.com:",
			"203.0.113.300",
			" origin.example.com",
		} {
			diags := resource.Validate(terraform.NewResourceConfigRaw(sourceGroupConfig(map[string]interface{}{"source": source})))
			require.Truef(t, diags.HasError(), "source %q must be rejected", source)
		}
	})

	t.Run("a config generated for a migrated group needs mode and max_fails set by hand", func(t *testing.T) {
		t.Parallel()

		diags := resource.Validate(terraform.NewResourceConfigRaw(sourceGroupConfig(
			map[string]interface{}{"source": "203.0.113.10", "mode": "", "max_fails": 0},
		)))
		support.RequireErrorDiagContains(t, diags, "source.0.mode")
		support.RequireErrorDiagContains(t, diags, "source.0.max_fails")
	})

	t.Run("text the api would trim is rejected", func(t *testing.T) {
		t.Parallel()

		config := sourceGroupConfig(map[string]interface{}{"source": "203.0.113.10", "comment": "main "})
		config["name"] = " origins"
		config["description"] = "main origins\n"

		diags := resource.Validate(terraform.NewResourceConfigRaw(config))
		support.RequireErrorDiagContains(t, diags, `expected name to have no leading or trailing whitespace (the API trims it), got " origins"`)
		support.RequireErrorDiagContains(t, diags, `expected description to have no leading or trailing whitespace (the API trims it), got "main origins\n"`)
		support.RequireErrorDiagContains(t, diags, `expected source.0.comment to have no leading or trailing whitespace (the API trims it), got "main "`)
	})
}

func TestIntegrationWebsecSourceGroup_Import(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	t.Run("a group id is imported without an api call", func(t *testing.T) {
		t.Parallel()

		imported, err := runImport(t, resource, websecmock.NewMockedWebsec(), testSourceGroupIDStr)
		require.NoError(t, err)
		require.Equal(t, testSourceGroupIDStr, imported.Id())
	})

	t.Run("resource:<resource_id> imports the group the resource uses", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).
			Return(&resources.Resource{ID: testResourceID, SourceGroup: ptr(testSourceGroupID)}, nil).Once()

		imported, err := runImport(t, resource, mc, "resource:"+testResourceIDStr)
		require.NoError(t, err)
		require.Equal(t, testSourceGroupIDStr, imported.Id())
	})

	t.Run("a resource without a source group has nothing to import", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).Return(&resources.Resource{ID: testResourceID}, nil).Once()

		_, err := runImport(t, resource, mc, "resource:"+testResourceIDStr)
		require.EqualError(t, err, "resource 42 has no source group")
	})

	t.Run("a malformed id is rejected with the expected format", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "abc", "0", "-7", "7:8", "resource:", "resource:0", "resources:42"} {
			_, err := runImport(t, resource, websecmock.NewMockedWebsec(), id)
			require.ErrorContainsf(t, err, "must be <group_id> or resource:<resource_id>", "import id %q", id)
		}
	})
}
