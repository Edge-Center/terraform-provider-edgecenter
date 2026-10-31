//go:build integration

package websec_test

import (
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/locations"
	"github.com/Edge-Center/edgecentercdn-go/websec/options"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
)

const (
	testLocationID    int64 = 5
	testLocationIDStr       = "5"
)

func locationConfig(extras ...map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		"resource_id": int(testResourceID),
		"name":        "static",
		"location":    "/static/",
	}
	for _, extra := range extras {
		maps.Copy(config, extra)
	}

	return config
}

func locationFromAPI() *locations.Location {
	return &locations.Location{
		ID:       testLocationID,
		Name:     "static",
		Location: "/static/",
		Active:   true,
		Weight:   1,
		Options: &options.LocationOptions{
			AllowedHTTPMethods:    options.Null[options.ListOption](),
			ASNACL:                options.Null[options.ASNACLOption](),
			RefererACL:            options.Null[options.AccessPolicyOption](),
			UserAgentACL:          options.Null[options.AccessPolicyOption](),
			AntibotFloodChallenge: options.Null[options.AntibotFloodChallengeOption](),
		},
	}
}

func TestIntegrationWebsecLocation_Create(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	var minimal, configured *locations.CreateRequest

	unavailable := serverError(t)
	disabled := apiError(t, http.StatusForbidden, cdnsdk.ErrForbidden,
		`{"message": "Resource is disabled. You do not have permission to create or edit rules for it. Please contact support team."}`)

	configuredLocation := locationFromAPI()
	configuredLocation.Active = false
	configuredLocation.Weight = 10
	configuredLocation.SourceGroup = ptr(testSourceGroupID)
	configuredLocation.Options.ASNACL = options.Value(options.ASNACLOption{
		Enabled:        true,
		PolicyType:     options.PolicyTypeDeny,
		ExceptedValues: []int64{64512},
	})

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create of a minimal location sends null for every option so that no api default applies",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Create", mock.Anything, testResourceID, mock.Anything).
					Run(func(args mock.Arguments) { minimal = args.Get(2).(*locations.CreateRequest) }).
					Return(locationFromAPI(), nil).Once()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(locationFromAPI(), nil).Once()

				return mc
			},
			NewConfig: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{
					"name": "static",
					"location": "/static/",
					"active": true,
					"weight": 1,
					"options": {
						"allowed_http_methods": null,
						"asn_acl": null,
						"referer_acl": null,
						"user_agent_acl": null,
						"antibot_flood_challenge": null
					}
				}`, requestJSON(t, minimal))

				support.RequireStateID(t, state, testLocationIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"resource_id":  testResourceIDStr,
					"name":         "static",
					"location":     "/static/",
					"active":       "true",
					"weight":       "1",
					"source_group": "0",
					"options.#":    "0",
				})
			},
		},
		{
			Name: "create sends the configured source group, weight, active and options",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Create", mock.Anything, testResourceID, mock.Anything).
					Run(func(args mock.Arguments) { configured = args.Get(2).(*locations.CreateRequest) }).
					Return(configuredLocation, nil).Once()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(configuredLocation, nil).Once()

				return mc
			},
			NewConfig: locationConfig(
				map[string]interface{}{"active": false, "weight": 10, "source_group": int(testSourceGroupID)},
				optionsConfig(map[string]interface{}{"asn_acl": policyOption("deny", 64512)}),
			),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{
					"name": "static",
					"location": "/static/",
					"active": false,
					"source_group": 7,
					"weight": 10,
					"options": {
						"allowed_http_methods": null,
						"asn_acl": {"enabled": true, "policy_type": "deny", "excepted_values": [64512]},
						"referer_acl": null,
						"user_agent_acl": null,
						"antibot_flood_challenge": null
					}
				}`, requestJSON(t, configured))

				support.RequireStateAttrs(t, state, map[string]string{
					"active":                                "false",
					"weight":                                "10",
					"source_group":                          testSourceGroupIDStr,
					"options.#":                             "1",
					"options.0.asn_acl.0.policy_type":       "deny",
					"options.0.asn_acl.0.excepted_values.#": "1",
					"options.0.referer_acl.#":               "0",
				})
			},
		},
		{
			Name: "create keeps the id when the read after it fails",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Create", mock.Anything, testResourceID, mock.Anything).Return(locationFromAPI(), nil).Once()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(nil, unavailable).Once()

				return mc
			},
			NewConfig: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testLocationIDStr)
			},
		},
		{
			Name: "a create the api refuses for a resource the administrator disabled shows the api error and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Create", mock.Anything, testResourceID, mock.Anything).Return(nil, disabled).Once()

				return mc
			},
			NewConfig: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Resource is disabled.")
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLocation_Read(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	remote := locationFromAPI()
	remote.Name = "assets"
	remote.Location = "^/assets/"
	remote.Active = false
	remote.Weight = 10
	remote.SourceGroup = ptr(testSourceGroupID)
	remote.Options.ASNACL = options.Value(options.ASNACLOption{
		Enabled:        true,
		PolicyType:     options.PolicyTypeDeny,
		ExceptedValues: []int64{64512, 64513},
	})
	remote.Options.UserAgentACL = options.Value(options.AccessPolicyOption{
		PolicyType:     options.PolicyTypeAllow,
		ExceptedValues: []string{"curl/8.0"},
	})

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "read maps the api answer into state and skips the options the api holds as null",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(remote, nil).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"resource_id":                            testResourceIDStr,
					"name":                                   "assets",
					"location":                               "^/assets/",
					"active":                                 "false",
					"weight":                                 "10",
					"source_group":                           testSourceGroupIDStr,
					"options.#":                              "1",
					"options.0.asn_acl.0.enabled":            "true",
					"options.0.asn_acl.0.policy_type":        "deny",
					"options.0.asn_acl.0.excepted_values.#":  "2",
					"options.0.user_agent_acl.0.enabled":     "false",
					"options.0.user_agent_acl.0.policy_type": "allow",
					"options.0.referer_acl.#":                "0",
					"options.0.allowed_http_methods.#":       "0",
					"options.0.antibot_flood_challenge.#":    "0",
				})
			},
		},
		{
			Name: "read removes a location the api cannot find from state",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
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
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(nil, serverError(t)).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testLocationIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLocation_Update(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	var sent *locations.UpdateRequest

	withGroup := map[string]interface{}{"source_group": int(testSourceGroupID)}

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "update sends null for a source group the config leaves out and no options when none changed",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Update", mock.Anything, testResourceID, testLocationID, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(3).(*locations.UpdateRequest) }).
					Return(locationFromAPI(), nil).Once()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(locationFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(withGroup),
			NewConfig:    locationConfig(map[string]interface{}{"name": "assets"}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"name": "assets", "location": "/static/", "active": true, "source_group": null, "weight": 1}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update sends the source group on every put, only the options that changed and null for a removed one",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Update", mock.Anything, testResourceID, testLocationID, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(3).(*locations.UpdateRequest) }).
					Return(locationFromAPI(), nil).Once()
				mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(locationFromAPI(), nil).Once()

				return mc
			},
			CurrentID: testLocationIDStr,
			CurrentState: locationConfig(withGroup, optionsConfig(map[string]interface{}{
				"asn_acl":        policyOption("deny", 64512),
				"referer_acl":    policyOption("allow", "example.com"),
				"user_agent_acl": policyOption("deny", "curl/8.0"),
			})),
			NewConfig: locationConfig(withGroup, optionsConfig(map[string]interface{}{
				"asn_acl":        policyOption("deny", 64513),
				"user_agent_acl": policyOption("deny", "curl/8.0"),
			})),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{
					"name": "static",
					"location": "/static/",
					"active": true,
					"source_group": 7,
					"weight": 1,
					"options": {
						"asn_acl": {"enabled": true, "policy_type": "deny", "excepted_values": [64513]},
						"referer_acl": null
					}
				}`, requestJSON(t, sent))
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLocation_Delete(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "delete removes the location from state",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Delete", mock.Anything, testResourceID, testLocationID).Return(nil).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete treats a 404 from the delete call as success",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Delete", mock.Anything, testResourceID, testLocationID).Return(notFound()).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete keeps the state when the api fails",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Locations.On("Delete", mock.Anything, testResourceID, testLocationID).Return(serverError(t)).Once()

				return mc
			},
			CurrentID:    testLocationIDStr,
			CurrentState: locationConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testLocationIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecLocation_Validate(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	tests := []struct {
		name   string
		config map[string]interface{}
		valid  bool
	}{
		{"a path", map[string]interface{}{"location": "/static/"}, true},
		{"a regular expression from the start of the path", map[string]interface{}{"location": `^/api/v[0-9]+/.*\.json$`}, true},
		{"a pattern of 255 characters", map[string]interface{}{"location": "/" + strings.Repeat("a", 254)}, true},
		{"a pattern of 256 characters", map[string]interface{}{"location": "/" + strings.Repeat("a", 255)}, false},
		{"a pattern without the leading slash", map[string]interface{}{"location": "static/"}, false},
		{"a caret without a slash after it", map[string]interface{}{"location": "^static/"}, false},
		{"a pattern of slashes only", map[string]interface{}{"location": "//"}, false},
		{"a caret and a slash only", map[string]interface{}{"location": "^/"}, false},
		{"a pattern with a space", map[string]interface{}{"location": "/static files/"}, false},
		{"a pattern with a quote", map[string]interface{}{"location": "/static'/"}, false},
		{"a pattern with a double quote", map[string]interface{}{"location": `/static"/`}, false},
		{"a pattern with a semicolon", map[string]interface{}{"location": "/static;/"}, false},
		{"a name the api would trim", map[string]interface{}{"name": "static "}, false},
		{"a name of 256 characters", map[string]interface{}{"name": strings.Repeat("a", 256)}, false},
		{"the largest weight", map[string]interface{}{"weight": 2147483647}, true},
		{"a weight above the int32 range", map[string]interface{}{"weight": 2147483648}, false},
		{"a zero weight", map[string]interface{}{"weight": 0}, false},
		{"a zero source group", map[string]interface{}{"source_group": 0}, false},
		{"a zero resource id", map[string]interface{}{"resource_id": 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diags := resource.Validate(terraform.NewResourceConfigRaw(locationConfig(tt.config)))
			require.Equal(t, tt.valid, !diags.HasError(), "%v", diags)
		})
	}
}

func TestIntegrationWebsecLocation_Import(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	t.Run("the import id sets resource_id, which the read that follows needs", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(locationFromAPI(), nil).Once()

		imported, err := runImport(t, resource, mc, testResourceIDStr+":"+testLocationIDStr)
		require.NoError(t, err)

		support.RequireNoDiags(t, resource.ReadContext(t.Context(), imported, mc.TestMeta()))
		support.RequireStateID(t, imported.State(), testLocationIDStr)
		support.RequireStateAttrs(t, imported.State(), map[string]string{
			"resource_id": testResourceIDStr,
			"name":        "static",
			"location":    "/static/",
		})
	})

	t.Run("a malformed id is rejected with the expected format", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "5", "42:", ":5", "42:abc", "0:5", "42:0", "-42:5", "42:5:1", "resource:42"} {
			_, err := runImport(t, resource, websecmock.NewMockedWebsec(), id)
			require.ErrorContainsf(t, err, "must be <resource_id>:<location_id>", "import id %q", id)
		}
	})
}
