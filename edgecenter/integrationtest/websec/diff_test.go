//go:build integration

package websec_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Edge-Center/edgecentercdn-go/websec/options"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
)

func planDiff(
	t *testing.T,
	resource *schema.Resource,
	current *terraform.InstanceState,
	config map[string]interface{},
	unknown ...cty.Path,
) (*terraform.InstanceDiff, error) {
	t.Helper()

	rawConfig, err := cty.Transform(support.NewRawConfig(t, resource, config), func(path cty.Path, value cty.Value) (cty.Value, error) {
		if slices.ContainsFunc(unknown, path.Equals) {
			return cty.UnknownVal(value.Type()), nil
		}

		return value, nil
	})
	require.NoError(t, err)

	if current == nil {
		current = &terraform.InstanceState{}
	}
	current.RawConfig = rawConfig

	return resource.Diff(t.Context(), current, terraform.NewResourceConfigShimmed(rawConfig, resource.CoreConfigSchema()), nil)
}

func requirePlanDiff(
	t *testing.T,
	resource *schema.Resource,
	current *terraform.InstanceState,
	config map[string]interface{},
	unknown ...cty.Path,
) *terraform.InstanceDiff {
	t.Helper()

	diff, err := planDiff(t, resource, current, config, unknown...)
	require.NoError(t, err)

	return diff
}

func TestIntegrationWebsecResource_Diff(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_resource")

	t.Run("a resource read with every option null plans no change for a config without options", func(t *testing.T) {
		t.Parallel()

		var nullOptions options.ResourceOptions
		require.NoError(t, json.Unmarshal([]byte(everyResourceOptionNull), &nullOptions))

		mc := websecmock.NewMockedWebsec()
		t.Cleanup(func() { mc.MockCleanup(t) })
		mc.Resources.On("Get", mock.Anything, testResourceID).
			Return(siteFromAPI(func(remote *resources.Resource) { remote.Options = &nullOptions }), nil)

		data := resource.Data(support.NewState(t, resource, siteConfig(), testResourceIDStr))
		support.RequireNoDiags(t, resource.ReadContext(t.Context(), data, mc.TestMeta()))

		diff := requirePlanDiff(t, resource, data.State(), siteConfig())
		require.True(t, diff.Empty(), "unexpected diff: %v", diff)
	})

	for name, current := range map[string]map[string]interface{}{
		"ssl_automated": {
			"ssl_certificate":           int(siteAutomatedCertID),
			"ssl_automated":             true,
			"ssl_certificate_automated": true,
		},
		"only the automated flag of the certificate": {
			"ssl_certificate":           int(siteAutomatedCertID),
			"ssl_certificate_automated": true,
		},
	} {
		t.Run("a config without ssl_certificate plans no change for a resource on an automated certificate with "+name, func(t *testing.T) {
			t.Parallel()

			diff := requirePlanDiff(t, resource, support.NewState(t, resource, siteConfig(current), testResourceIDStr), siteConfig())
			require.True(t, diff.Empty(), "unexpected diff: %v", diff)
		})
	}

	t.Run("another domain replaces the resource", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t,
			resource,
			support.NewState(t, resource, siteConfig(), testResourceIDStr),
			siteConfig(map[string]interface{}{"domain": "example.org", "subdomains": []interface{}{"www.example.org"}}),
		)
		require.True(t, diff.RequiresNew())
	})

	t.Run("every other attribute is updated in place", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t,
			resource,
			support.NewState(t, resource, siteConfig(map[string]interface{}{"ssl_certificate": int(testCertID)}), testResourceIDStr),
			siteConfig(
				map[string]interface{}{
					"source_group":    int(testSourceGroupID) + 1,
					"ssl_certificate": int(siteNewCertID),
					"subdomains":      []interface{}{siteWWW, siteAPI},
					"active":          false,
				},
				siteTLSAndWAF(true),
			),
		)
		require.False(t, diff.Empty())
		require.False(t, diff.RequiresNew())
	})

	t.Run("an own certificate removed from the config plans its removal", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, siteConfig(map[string]interface{}{"ssl_certificate": int(testCertID)}), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, siteConfig())
		require.Contains(t, diff.Attributes, "ssl_certificate")
	})

	t.Run("a certificate reference unknown until apply stays in the plan of a resource on an automated certificate", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, siteConfig(map[string]interface{}{
			"ssl_certificate":           int(siteAutomatedCertID),
			"ssl_automated":             true,
			"ssl_certificate_automated": true,
		}), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, siteConfig(), cty.GetAttrPath("ssl_certificate"))
		require.Contains(t, diff.Attributes, "ssl_certificate")
		require.True(t, diff.Attributes["ssl_certificate"].NewComputed)
	})

	t.Run("an address in the config plans no change against its network in state", func(t *testing.T) {
		t.Parallel()

		blacklist := func(value string) map[string]interface{} {
			return optionsConfig(map[string]interface{}{
				"blacklist": option(map[string]interface{}{"value": []interface{}{value}}),
			})
		}

		current := support.NewState(t, resource, siteConfig(blacklist("9.9.9.9/32")), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, siteConfig(blacklist("9.9.9.9")))
		require.True(t, diff.Empty(), "unexpected diff: %v", diff)
	})

	t.Run("the rules on subdomains wait while the domain is unknown", func(t *testing.T) {
		t.Parallel()

		config := siteConfig(
			map[string]interface{}{"subdomains": []interface{}{siteWWW, "api.example.org"}},
			optionsConfig(map[string]interface{}{
				"www_redirects": option(map[string]interface{}{"value": true}),
			}),
		)

		_, err := planDiff(t, resource, nil, config, cty.GetAttrPath("domain"))
		require.NoError(t, err)
	})

	header := func(name, value string) map[string]interface{} {
		return map[string]interface{}{"name": name, "value": []interface{}{value}}
	}

	refused := map[string]struct {
		config map[string]interface{}
		want   string
	}{
		"an empty options block": {
			config: siteConfig(optionsConfig(map[string]interface{}{})),
			want:   "the options block is empty",
		},
		"a subdomain of another domain": {
			config: siteConfig(map[string]interface{}{"subdomains": []interface{}{siteWWW, "www.example.org"}}),
			want:   `subdomain "www.example.org" does not end with ".example.com"`,
		},
		"www_redirects without the www subdomain": {
			config: siteConfig(
				map[string]interface{}{"subdomains": []interface{}{siteAPI}},
				optionsConfig(map[string]interface{}{"www_redirects": option(map[string]interface{}{"value": true})}),
			),
			want: `options.0.www_redirects needs "www.example.com" in subdomains`,
		},
		"two response headers whose names differ only in case": {
			config: siteConfig(optionsConfig(map[string]interface{}{
				"response_headers": option(map[string]interface{}{
					"value": []interface{}{header("X-Cache", "HIT"), header("x-cache", "MISS")},
				}),
			})),
			want: `more than one "x-cache" header`,
		},
	}

	for name, tc := range refused {
		t.Run("the plan refuses "+name, func(t *testing.T) {
			t.Parallel()

			_, err := planDiff(t, resource, nil, tc.config)
			require.ErrorContains(t, err, tc.want)
		})
	}

	t.Run("the plan accepts a disabled www_redirects without the www subdomain", func(t *testing.T) {
		t.Parallel()

		config := siteConfig(
			map[string]interface{}{"subdomains": []interface{}{siteAPI}},
			optionsConfig(map[string]interface{}{
				"www_redirects": option(map[string]interface{}{"enabled": false, "value": true}),
			}),
		)

		_, err := planDiff(t, resource, nil, config)
		require.NoError(t, err)
	})

	disabledWWWRedirects := optionsConfig(map[string]interface{}{
		"www_redirects": option(map[string]interface{}{"enabled": false, "value": false}),
	})
	withoutWWW := map[string]interface{}{"subdomains": []interface{}{siteAPI}}

	t.Run("the plan refuses to remove the www subdomain while the www_redirects block stays, since the api deletes the option with it", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, siteConfig(disabledWWWRedirects), testResourceIDStr)

		_, err := planDiff(t, resource, current, siteConfig(disabledWWWRedirects, withoutWWW))
		require.ErrorContains(t, err, `removing "www.example.com" from subdomains deletes options.0.www_redirects in the API`)
	})

	t.Run("the www subdomain goes together with the www_redirects block", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, siteConfig(disabledWWWRedirects), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, siteConfig(withoutWWW))
		require.False(t, diff.Empty())
	})
}

func TestIntegrationWebsecSourceGroup_Diff(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_source_group")

	t.Run("a source listed twice is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, sourceGroupConfig(
			map[string]interface{}{"source": "203.0.113.10", "weight": 30},
			map[string]interface{}{"source": "203.0.113.10", "weight": 30},
		))
		require.ErrorContains(t, err, `source "203.0.113.10" is listed more than once`)
	})

	t.Run("sources unknown until apply are not taken for duplicates", func(t *testing.T) {
		t.Parallel()

		config := sourceGroupConfig(
			map[string]interface{}{"source": "203.0.113.10"},
			map[string]interface{}{"source": "203.0.113.11"},
		)

		_, err := planDiff(t, resource, nil, config,
			cty.GetAttrPath("source").IndexInt(0).GetAttr("source"),
			cty.GetAttrPath("source").IndexInt(1).GetAttr("source"),
		)
		require.NoError(t, err, "an unknown source reads as an empty string, two of them must not look equal")
	})

	t.Run("a third source with the default weight breaks the weight limit whatever its mode", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, sourceGroupConfig(
			map[string]interface{}{"source": "203.0.113.10"},
			map[string]interface{}{"source": "203.0.113.11", "mode": "backup"},
			map[string]interface{}{"source": "203.0.113.12", "mode": "down"},
		))
		require.ErrorContains(t, err, "the weights of all sources add up to 150, but a source group allows at most 100")
	})

	t.Run("weights that add up to exactly 100 pass", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, sourceGroupConfig(
			map[string]interface{}{"source": "203.0.113.10", "weight": 60},
			map[string]interface{}{"source": "203.0.113.11", "weight": 40, "mode": "backup"},
		))
		require.NoError(t, err)
	})

	t.Run("a change of every attribute is an in place update", func(t *testing.T) {
		t.Parallel()

		config := fullSourceGroupConfig()
		config["name"] = "renamed origins"

		diff := requirePlanDiff(t, resource, support.NewState(t, resource, minimalSourceGroupConfig(), testSourceGroupIDStr), config)
		require.False(t, diff.Empty())
		require.False(t, diff.RequiresNew(), "a replaced group would hit the in use check on delete, so nothing may force it")
	})
}

func TestIntegrationWebsecSSLCertificate_Diff(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	created := func(t *testing.T) *terraform.InstanceState {
		t.Helper()

		return support.NewState(t, resource, sslCertConfig(), testCertIDStr)
	}

	changes := map[string]map[string]interface{}{
		"a new certificate": sslCertConfigWith("certificate", sslCertPEMBlock("CERTIFICATE", "renewed leaf")),
		"a new private key": sslCertConfigWith("private_key", sslCertPEMBlock("PRIVATE KEY", "renewed key")),
		"a new chain":       sslCertConfigWith("certificates_chain", sslCertPEMBlock("CERTIFICATE", "renewed intermediate")),
		"a removed chain":   sslCertConfigWithout("certificates_chain"),
	}

	for name, config := range changes {
		t.Run(name+" replaces a certificate terraform created", func(t *testing.T) {
			t.Parallel()

			diff := requirePlanDiff(t, resource, created(t), config)
			require.True(t, diff.RequiresNew())
		})
	}

	t.Run("a chain added to a certificate created without one replaces it", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, sslCertConfigWithout("certificates_chain"), testCertIDStr)

		diff := requirePlanDiff(t, resource, current, sslCertConfig())
		require.True(t, diff.RequiresNew())
		require.True(t, diff.Attributes["certificates_chain"].RequiresNew)
	})

	t.Run("a certificate unknown until apply replaces a certificate terraform created", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t, resource, created(t), sslCertConfig(), cty.GetAttrPath("certificate"))
		require.True(t, diff.RequiresNew())
		require.True(t, diff.Attributes["certificate"].NewComputed)
	})

	t.Run("a rename updates the certificate in place", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t, resource, created(t), sslCertConfigWith("name", sslCertRenamed))
		require.False(t, diff.RequiresNew())
		require.Equal(t, sslCertRenamed, diff.Attributes["name"].New)
	})

	t.Run("the first plan after import takes the material in place", func(t *testing.T) {
		t.Parallel()

		imported := support.NewState(t, resource, map[string]interface{}{"name": sslCertName}, testCertIDStr)

		diff := requirePlanDiff(t, resource, imported, sslCertConfig())
		require.False(t, diff.RequiresNew(), "an imported certificate has no material in state, so the first plan must not replace it")
		for _, key := range []string{"certificate", "private_key", "certificates_chain"} {
			require.Contains(t, diff.Attributes, key)
		}
	})
}

func TestIntegrationWebsecLocation_Diff(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_location")

	t.Run("a location read with every option null and no source group plans no change for a config without them", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		t.Cleanup(func() { mc.MockCleanup(t) })
		mc.Locations.On("Get", mock.Anything, testResourceID, testLocationID).Return(locationFromAPI(), nil).Once()

		data := resource.Data(support.NewState(t, resource, locationConfig(), testLocationIDStr))
		support.RequireNoDiags(t, resource.ReadContext(t.Context(), data, mc.TestMeta()))

		diff := requirePlanDiff(t, resource, data.State(), locationConfig())
		require.True(t, diff.Empty(), "unexpected diff: %v", diff)
	})

	t.Run("an empty options block is refused", func(t *testing.T) {
		t.Parallel()

		_, err := planDiff(t, resource, nil, locationConfig(optionsConfig(map[string]interface{}{})))
		require.ErrorContains(t, err, "the options block is empty")
	})

	t.Run("another resource_id replaces the location", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t,
			resource,
			support.NewState(t, resource, locationConfig(), testLocationIDStr),
			locationConfig(map[string]interface{}{"resource_id": int(testResourceID) + 1}),
		)
		require.True(t, diff.RequiresNew())
	})

	t.Run("every other attribute is updated in place", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t,
			resource,
			support.NewState(t, resource, locationConfig(), testLocationIDStr),
			locationConfig(
				map[string]interface{}{
					"name":         "assets",
					"location":     "^/assets/",
					"active":       false,
					"weight":       10,
					"source_group": int(testSourceGroupID),
				},
				optionsConfig(map[string]interface{}{"asn_acl": policyOption("deny", 64512)}),
			),
		)
		require.False(t, diff.Empty())
		require.False(t, diff.RequiresNew())
	})
}

func TestIntegrationWebsecLECert_Diff(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_lecert")

	t.Run("changing cert_type replaces the certificate", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, leCertState(), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, map[string]interface{}{"resource_id": int(testResourceID), "cert_type": "MDDC"})
		require.True(t, diff.RequiresNew())
	})

	t.Run("another resource_id replaces the certificate", func(t *testing.T) {
		t.Parallel()

		current := support.NewState(t, resource, leCertState(), testResourceIDStr)

		diff := requirePlanDiff(t, resource, current, map[string]interface{}{"resource_id": int(testResourceID) + 1})
		require.True(t, diff.RequiresNew())
	})

	importLECert := func(t *testing.T, mc *websecmock.MockedWebsec) *terraform.InstanceState {
		t.Helper()

		imported, err := runImport(t, resource, mc, testResourceIDStr)
		require.NoError(t, err)
		support.RequireNoDiags(t, resource.ReadContext(t.Context(), imported, mc.TestMeta()))

		return imported.State()
	}

	importMoved := func(t *testing.T) *terraform.InstanceState {
		t.Helper()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID)), nil).Once()
		mc.LECerts.On("Get", mock.Anything, testResourceID).Return(nil, notFound()).Once()
		mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(&sslcerts.Cert{ID: testCertID, Automated: true}, nil).Once()

		return importLECert(t, mc)
	}

	t.Run("a moved certificate without a request on record imports and plans no change for the default cert_type", func(t *testing.T) {
		t.Parallel()

		state := importMoved(t)
		support.RequireStateAttrs(t, state, map[string]string{
			"resource_id":     testResourceIDStr,
			"cert_type":       "",
			"ssl_certificate": testCertIDStr,
		})

		diff := requirePlanDiff(t, resource, state, leCertConfig())
		require.True(t, diff.Empty(), "the default cert_type must not replace a certificate whose type the api does not report")
	})

	t.Run("another cert_type replaces a moved certificate without a request on record", func(t *testing.T) {
		t.Parallel()

		diff := requirePlanDiff(t, resource, importMoved(t), map[string]interface{}{"resource_id": int(testResourceID), "cert_type": "MDDC"})
		require.True(t, diff.RequiresNew())
	})

	t.Run("an imported certificate takes the type the api reports and another cert_type replaces it", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).Return(siteFromAPI(siteCertificate(testCertID), siteSSLAutomated), nil).Once()
		mc.LECerts.On("Get", mock.Anything, testResourceID).Return(leCertIssued(), nil).Once()

		state := importLECert(t, mc)
		support.RequireStateAttrs(t, state, map[string]string{"cert_type": "LE"})

		diff := requirePlanDiff(t, resource, state, map[string]interface{}{"resource_id": int(testResourceID), "cert_type": "EC"})
		require.True(t, diff.RequiresNew())
	})
}
