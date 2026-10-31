//go:build integration

package websec_test

import (
	"maps"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/options"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
)

const (
	siteDomain      = "example.com"
	siteWWW         = "www.example.com"
	siteAPI         = "api.example.com"
	siteProtectedIP = "203.0.113.10"
	siteClientID    = 1188013

	siteAutomatedCertID int64 = 9
	siteNewCertID       int64 = 12

	everyResourceOptionNull = `{
		"use_default_le_chain": null, "use_rsa_le_cert": null, "allowed_http_methods": null, "asn_acl": null,
		"referer_acl": null, "user_agent_acl": null, "antibot_flood_challenge": null, "blacklist": null,
		"whitelist": null, "geo_acl": null, "ip_hash": null, "redirect_http_to_https": null, "response_headers": null,
		"tls_versions": null, "country_acl": null, "www_redirects": null, "source_protocol_https": null,
		"upstream_healthchecker": null, "waf": null, "ml_analysis": null, "waf_monitoring_mode": null,
		"redirect_https_to_http": null, "antibot_challenge_url_exceptions": null,
		"antibot_flood_challenge_user_agent_whitelist": null
	}`

	subdomainsLockedBody = `{"message": {"subdomains": "The subdomains deletion operation is temporarily blocked by another process. Please try again later."}}`
)

func siteConfig(extras ...map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		"domain":       siteDomain,
		"source_group": int(testSourceGroupID),
		"subdomains":   []interface{}{siteWWW},
	}
	for _, extra := range extras {
		maps.Copy(config, extra)
	}

	return config
}

func siteTLSAndWAF(waf bool) map[string]interface{} {
	return optionsConfig(map[string]interface{}{
		"tls_versions": option(map[string]interface{}{"value": []interface{}{"TLSv1.3"}}),
		"waf":          option(map[string]interface{}{"value": waf}),
	})
}

func siteFromAPI(changes ...func(*resources.Resource)) *resources.Resource {
	remote := &resources.Resource{
		ID:          testResourceID,
		Domain:      siteDomain,
		Subdomains:  []string{siteWWW},
		Active:      true,
		Enabled:     true,
		Status:      resources.StatusProcessed,
		ProtectedIP: ptr(siteProtectedIP),
		SourceGroup: ptr(testSourceGroupID),
		Client:      siteClientID,
	}
	for _, change := range changes {
		change(remote)
	}

	return remote
}

func siteCertificate(id int64) func(*resources.Resource) {
	return func(remote *resources.Resource) { remote.SSLCertificate = ptr(id) }
}

func siteInactive(remote *resources.Resource) {
	remote.Active = false
}

func siteSSLAutomated(remote *resources.Resource) {
	remote.SSLAutomated = true
}

func TestIntegrationWebsecResource_Create(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	var sent *resources.CreateRequest

	tls := options.Value(options.ListOption{Enabled: true, Value: []string{"TLSv1.3"}})
	created := siteFromAPI(siteInactive, siteCertificate(testCertID), func(remote *resources.Resource) {
		remote.Options = &options.ResourceOptions{TLSVersions: tls}
	})

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create without an options block sends null for every option so that no api default applies",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Create", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(1).(*resources.CreateRequest) }).
					Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()

				return mc
			},
			NewConfig: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{
					"domain": "example.com",
					"subdomains": ["www.example.com"],
					"active": true,
					"source_group": 7,
					"options": `+everyResourceOptionNull+`
				}`, requestJSON(t, sent))

				support.RequireStateID(t, state, testResourceIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"protected_ip":              siteProtectedIP,
					"status":                    "processed",
					"enabled":                   "true",
					"client":                    "1188013",
					"ssl_certificate":           "0",
					"ssl_certificate_automated": "false",
					"options.#":                 "0",
				})
			},
		},
		{
			Name: "create sends the configured options, active false and the certificate, but never ssl_automated",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Create", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) { sent = args.Get(1).(*resources.CreateRequest) }).
					Return(created, nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(created, nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID}, nil).Once()

				return mc
			},
			NewConfig: siteConfig(
				map[string]interface{}{"active": false, "ssl_certificate": int(testCertID)},
				optionsConfig(map[string]interface{}{
					"tls_versions": option(map[string]interface{}{"value": []interface{}{"TLSv1.3"}}),
				}),
			),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Equal(t, ptr(false), sent.Active)
				require.Equal(t, ptr(testCertID), sent.SSLCertificate)
				require.NotContains(t, requestJSON(t, sent), "ssl_automated")
				require.Equal(t, tls, sent.Options.TLSVersions)
				require.True(t, sent.Options.WAF.IsNull())
				require.True(t, sent.Options.GeoACL.IsNull())

				support.RequireStateAttrs(t, state, map[string]string{
					"active":                           "false",
					"ssl_certificate":                  testCertIDStr,
					"ssl_certificate_automated":        "false",
					"options.#":                        "1",
					"options.0.tls_versions.0.enabled": "true",
					"options.0.tls_versions.0.value.#": "1",
					"options.0.waf.#":                  "0",
				})
			},
		},
		{
			Name: "create keeps the id when the read after it fails",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Create", mock.Anything, mock.Anything).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(nil, serverError(t)).Once()

				return mc
			},
			NewConfig: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "a rejected create shows the api error and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Create", mock.Anything, mock.Anything).Return(nil, apiError(t, http.StatusBadRequest,
					cdnsdk.ErrBadRequest, `{"errors": {"domain": ["ddos resource with this domain already exists."]}}`)).Once()

				return mc
			},
			NewConfig: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "domain: ddos resource with this domain already exists.")
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCaseWithRawConfig)
}

func TestIntegrationWebsecResource_Read(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "read stores the options of the schema the api holds and skips the others",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				remote := siteFromAPI(func(remote *resources.Resource) {
					remote.Options = &options.ResourceOptions{
						TLSVersions: options.Value(options.ListOption{Enabled: true, Value: []string{"TLSv1.2", "TLSv1.3"}}),
						WAF:         options.Null[options.BooleanOption](),
						GeoACL:      options.Value(options.GeoACLOption{PolicyType: options.PolicyTypeDeny}),
					}
				})

				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(remote, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"options.#":                        "1",
					"options.0.tls_versions.0.value.#": "2",
					"options.0.waf.#":                  "0",
				})
			},
		},
		{
			Name: "read leaves the options block out when the api holds no option of the schema",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				remote := siteFromAPI(func(remote *resources.Resource) {
					remote.Options = &options.ResourceOptions{
						GeoACL:       options.Value(options.GeoACLOption{PolicyType: options.PolicyTypeDeny}),
						UseRSALECert: options.Value(options.BooleanOption{Enabled: true, Value: true}),
					}
				})

				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(remote, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"options.#": "0"})
			},
		},
		{
			Name: "read takes the certificate of a resource with ssl_automated as automated without fetching it",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).
					Return(siteFromAPI(siteCertificate(siteAutomatedCertID), siteSSLAutomated), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_certificate":           "9",
					"ssl_automated":             "true",
					"ssl_certificate_automated": "true",
				})
			},
		},
		{
			Name: "read takes the automated flag from the linked certificate when ssl_automated is false",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(siteAutomatedCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, siteAutomatedCertID).
					Return(&sslcerts.Cert{ID: siteAutomatedCertID, Automated: true}, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_automated":             "false",
					"ssl_certificate_automated": "true",
				})
			},
		},
		{
			Name: "read takes a linked certificate the api cannot find as not automated",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(siteAutomatedCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, siteAutomatedCertID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"ssl_certificate":           "9",
					"ssl_certificate_automated": "false",
				})
			},
		},
		{
			Name: "read removes a resource the api cannot find from state",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecResource_Update(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	var sent *resources.UpdateRequest

	captureUpdate := func(args mock.Arguments) { sent = args.Get(2).(*resources.UpdateRequest) }
	redirect := optionsConfig(map[string]interface{}{
		"redirect_http_to_https": option(map[string]interface{}{"value": true}),
	})
	twoSubdomains := siteConfig(map[string]interface{}{"subdomains": []interface{}{siteWWW, siteAPI}})
	locked := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest, subdomainsLockedBody)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "update sends only the option that changed and leaves subdomains and the certificate out",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Twice()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(siteFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			NewConfig:    siteConfig(siteTLSAndWAF(false)),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": true, "source_group": 7, "options": {"waf": {"enabled": true, "value": false}}}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update deletes the option of a removed block with null",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Twice()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(siteFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			NewConfig: siteConfig(optionsConfig(map[string]interface{}{
				"tls_versions": option(map[string]interface{}{"value": []interface{}{"TLSv1.3"}}),
			})),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": true, "source_group": 7, "options": {"waf": null}}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update without a change of options sends no options at all",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(siteFromAPI(siteInactive), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteInactive), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(siteTLSAndWAF(true)),
			NewConfig:    siteConfig(siteTLSAndWAF(true), map[string]interface{}{"active": false}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": false, "source_group": 7}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update sends the whole set of subdomains when it changes",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				updated := siteFromAPI(func(remote *resources.Resource) { remote.Subdomains = []string{siteWWW, siteAPI} })

				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(updated, nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(updated, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			NewConfig:    twoSubdomains,
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)

				values, ok := sent.Subdomains.Get()
				require.True(t, ok)
				require.ElementsMatch(t, []string{siteWWW, siteAPI}, values)
				require.Nil(t, sent.Options)
				support.RequireStateAttrs(t, state, map[string]string{"subdomains.#": "2"})
			},
		},
		{
			Name: "update removes every subdomain with an empty list",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				updated := siteFromAPI(func(remote *resources.Resource) { remote.Subdomains = nil })

				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(updated, nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(updated, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			NewConfig:    siteConfig(map[string]interface{}{"subdomains": []interface{}{}}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"subdomains": [], "active": true, "source_group": 7}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update sends a new certificate with ssl_automated false and the redirect option the api checks it with",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID}, nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).
					Return(siteFromAPI(siteCertificate(siteNewCertID)), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(siteNewCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, siteNewCertID).Return(&sslcerts.Cert{ID: siteNewCertID}, nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(redirect, map[string]interface{}{"ssl_certificate": int(testCertID)}),
			NewConfig:    siteConfig(redirect, map[string]interface{}{"ssl_certificate": int(siteNewCertID)}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{
					"active": true,
					"ssl_certificate": 12,
					"ssl_automated": false,
					"source_group": 7,
					"options": {"redirect_http_to_https": {"enabled": true, "value": true}}
				}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update sends null for a removed certificate together with ssl_automated false",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID}, nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(map[string]interface{}{"ssl_certificate": int(testCertID)}),
			NewConfig:    siteConfig(),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": true, "ssl_certificate": null, "ssl_automated": false, "source_group": 7}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update clears the ssl_automated a revoke left behind even without a certificate change",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteSSLAutomated), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).Return(siteFromAPI(siteInactive), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteInactive), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(map[string]interface{}{"ssl_automated": true}),
			NewConfig:    siteConfig(map[string]interface{}{"active": false}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": false, "ssl_automated": false, "source_group": 7}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update leaves the automated certificate alone when the config does not set one",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).
					Return(siteFromAPI(siteCertificate(siteAutomatedCertID), siteSSLAutomated), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Run(captureUpdate).
					Return(siteFromAPI(siteCertificate(siteAutomatedCertID), siteSSLAutomated, siteInactive), nil).Once()
				mc.Resources.On("Get", mock.Anything, testResourceID).
					Return(siteFromAPI(siteCertificate(siteAutomatedCertID), siteSSLAutomated, siteInactive), nil).Once()

				return mc
			},
			CurrentID: testResourceIDStr,
			CurrentState: siteConfig(map[string]interface{}{
				"ssl_certificate":           int(siteAutomatedCertID),
				"ssl_automated":             true,
				"ssl_certificate_automated": true,
			}),
			NewConfig: siteConfig(map[string]interface{}{"active": false}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.JSONEq(t, `{"active": false, "source_group": 7}`, requestJSON(t, sent))
			},
		},
		{
			Name: "update refuses to switch the certificate while an automated certificate is linked",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(siteAutomatedCertID)), nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, siteAutomatedCertID).
					Return(&sslcerts.Cert{ID: siteAutomatedCertID, Automated: true}, nil).Once()

				return mc
			},
			CurrentID: testResourceIDStr,
			CurrentState: siteConfig(map[string]interface{}{
				"ssl_certificate":           int(siteAutomatedCertID),
				"ssl_certificate_automated": true,
			}),
			NewConfig: siteConfig(map[string]interface{}{"ssl_certificate": int(siteNewCertID)}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "remove the edgecenter_websec_lecert of this resource in the same apply")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "update refuses a resource without a protected ip, since any update would take a new one",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).
					Return(siteFromAPI(func(remote *resources.Resource) { remote.ProtectedIP = nil }), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			NewConfig:    siteConfig(map[string]interface{}{"active": false}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "has no protected IP")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
		{
			Name: "update retries while another request holds the lock on subdomain deletion",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Twice()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Return(nil, locked).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Return(siteFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: twoSubdomains,
			NewConfig:    siteConfig(),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
			},
		},
		{
			Name: "update returns the lock error of the api when the update timeout leaves no room for a retry",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).Return(nil, locked).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: twoSubdomains,
			NewConfig:    siteConfig(map[string]interface{}{"timeouts": map[string]interface{}{"update": "60s"}}),
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "temporarily blocked by another process")
			},
		},
		{
			Name: "update returns any other subdomain error at once",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(), nil).Once()
				mc.Resources.On("Update", mock.Anything, testResourceID, mock.Anything).
					Return(nil, apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest,
						`{"errors": {"subdomains": "Subdomain api.example.com already exists."}}`)).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			NewConfig:    twoSubdomains,
			Check: func(t *testing.T, _ *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Subdomain api.example.com already exists.")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCaseWithRawConfig)
}

func TestIntegrationWebsecResource_Delete(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "delete removes the resource from state",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.Resources.On("Delete", mock.Anything, testResourceID).Return(nil).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
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
				mc.Resources.On("Delete", mock.Anything, testResourceID).Return(notFound()).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
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
				mc.Resources.On("Delete", mock.Anything, testResourceID).Return(serverError(t)).Once()

				return mc
			},
			CurrentID:    testResourceIDStr,
			CurrentState: siteConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testResourceIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecResource_Validate(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	withOption := func(name string, fields map[string]interface{}) map[string]interface{} {
		return optionsConfig(map[string]interface{}{name: option(fields)})
	}
	withPolicy := func(name string, values ...interface{}) map[string]interface{} {
		return optionsConfig(map[string]interface{}{name: policyOption("deny", values...)})
	}

	tests := []struct {
		name   string
		config map[string]interface{}
		valid  bool
	}{
		{"a wildcard domain", map[string]interface{}{"domain": "*.example.com", "subdomains": []interface{}{}}, true},
		{"a punycode domain", map[string]interface{}{"domain": "xn--e1afmkfd.xn--p1ai", "subdomains": []interface{}{}}, true},
		{"a domain in uppercase", map[string]interface{}{"domain": "Example.com"}, false},
		{"a domain of one level", map[string]interface{}{"domain": "example"}, false},
		{"a wildcard subdomain", map[string]interface{}{"subdomains": []interface{}{"*.example.com"}}, true},
		{"a wildcard in the middle of a subdomain", map[string]interface{}{"subdomains": []interface{}{"a.*.example.com"}}, false},
		{"a subdomain in uppercase", map[string]interface{}{"subdomains": []interface{}{"WWW.example.com"}}, false},
		{"a referer in uppercase", withPolicy("referer_acl", "Example.com"), false},
		{"a country in lowercase", withPolicy("country_acl", "ru"), false},
		{"a country in uppercase", withPolicy("country_acl", "RU"), true},
		{"the largest ASN", withPolicy("asn_acl", 2147483647), true},
		{"an ASN above the int32 range", withPolicy("asn_acl", 2147483648), false},
		{"a zero ASN", withPolicy("asn_acl", 0), false},
		{"an unknown policy type", withOption("user_agent_acl", map[string]interface{}{"policy_type": "block", "excepted_values": []interface{}{"curl"}}), false},
		{"addresses and networks of both families", withOption("blacklist", map[string]interface{}{"value": []interface{}{"10.0.0.5/24", "2001:db8::1"}}), true},
		{"an invalid address", withOption("whitelist", map[string]interface{}{"value": []interface{}{"300.1.1.1"}}), false},
		{"a TLS version with a leading space", withOption("tls_versions", map[string]interface{}{"value": []interface{}{" TLSv1.2"}}), false},
		{"an unknown antibot challenge", withOption("antibot_flood_challenge", map[string]interface{}{"value": "CAPTCHA"}), false},
		{"an antibot challenge without a type", withOption("antibot_flood_challenge", map[string]interface{}{"enabled": false}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diags := resource.Validate(terraform.NewResourceConfigRaw(siteConfig(tt.config)))
			require.Equal(t, tt.valid, !diags.HasError(), "%v", diags)
		})
	}
}

func TestIntegrationWebsecResource_Import(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	t.Run("the resource id is imported without an api call", func(t *testing.T) {
		t.Parallel()

		imported, err := runImport(t, resource, websecmock.NewMockedWebsec(), testResourceIDStr)
		require.NoError(t, err)
		require.Equal(t, testResourceIDStr, imported.Id())
	})

	t.Run("a malformed id is rejected with the expected format", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "example.com", "0", "-42", "42:7", "resource:42"} {
			_, err := runImport(t, resource, websecmock.NewMockedWebsec(), id)
			require.ErrorContainsf(t, err, "must be <resource_id>", "import id %q", id)
		}
	})
}
