package websec

import (
	"encoding/json"
	"math"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/Edge-Center/edgecentercdn-go/websec/options"
)

func sortedJSON(t *testing.T, v interface{}) string {
	t.Helper()

	raw, err := json.Marshal(v)
	require.NoError(t, err)

	var tree interface{}
	require.NoError(t, json.Unmarshal(raw, &tree))

	sorted, err := json.Marshal(sortArrays(tree))
	require.NoError(t, err)

	return string(sorted)
}

func sortArrays(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			typed[key] = sortArrays(value)
		}
	case []interface{}:
		for i := range typed {
			typed[i] = sortArrays(typed[i])
		}

		sort.Slice(typed, func(i, j int) bool {
			left, _ := json.Marshal(typed[i])
			right, _ := json.Marshal(typed[j])

			return string(left) < string(right)
		})
	}

	return v
}

func locationOptionsWithEveryValue() options.LocationOptions {
	challenge := options.AntibotChallengeJSBox

	return options.LocationOptions{
		AllowedHTTPMethods: options.Value(options.ListOption{Enabled: true, Value: []string{"GET", "HEAD"}}),
		ASNACL: options.Value(options.ASNACLOption{
			Enabled:        true,
			PolicyType:     options.PolicyTypeDeny,
			ExceptedValues: []int64{64512, math.MaxInt32},
		}),
		RefererACL: options.Value(options.AccessPolicyOption{
			PolicyType:     options.PolicyTypeAllow,
			ExceptedValues: []string{"", "*.example.com"},
		}),
		UserAgentACL: options.Value(options.AccessPolicyOption{
			Enabled:        true,
			PolicyType:     options.PolicyTypeDeny,
			ExceptedValues: []string{"~bot", "curl/8.0"},
		}),
		AntibotFloodChallenge: options.Value(options.AntibotFloodChallengeOption{
			Enabled:     true,
			Value:       &challenge,
			TimeLimit:   5,
			GetOnlyOn:   true,
			UTMReferrer: true,
		}),
	}
}

func TestEveryResourceOptionSurvivesTheSchemaRoundTrip(t *testing.T) {
	t.Parallel()

	want := options.ResourceOptions{
		LocationOptions:     locationOptionsWithEveryValue(),
		UseDefaultLEChain:   options.Null[options.BooleanOption](),
		UseRSALECert:        options.Null[options.BooleanOption](),
		Blacklist:           options.Value(options.ListOption{Enabled: true, Value: []string{"198.51.100.0/24", "2001:db8::/32"}}),
		Whitelist:           options.Value(options.ListOption{Value: []string{"203.0.113.7/32"}}),
		GeoACL:              options.Null[options.GeoACLOption](),
		IPHash:              options.Value(options.BooleanOption{Enabled: true}),
		RedirectHTTPToHTTPS: options.Value(options.BooleanOption{Value: true}),
		ResponseHeaders: options.Value(options.ResponseHeadersOption{
			Enabled: true,
			Value: []options.ResponseHeader{
				{Name: "X-Frame-Options", Value: []string{"DENY"}},
				{Name: "Cache-Control", Value: []string{"no-cache", "no-store"}, Always: true},
			},
		}),
		TLSVersions: options.Value(options.ListOption{Enabled: true, Value: []string{"TLSv1.2", "TLSv1.3"}}),
		CountryACL: options.Value(options.AccessPolicyOption{
			Enabled:        true,
			PolicyType:     options.PolicyTypeAllow,
			ExceptedValues: []string{"RU", "KZ"},
		}),
		WWWRedirects:                            options.Value(options.BooleanOption{Enabled: true, Value: true}),
		SourceProtocolHTTPS:                     options.Value(options.BooleanOption{Enabled: true, Value: true}),
		UpstreamHealthChecker:                   options.Null[options.UpstreamHealthCheckerOption](),
		WAF:                                     options.Value(options.BooleanOption{Value: true}),
		MLAnalysis:                              options.Null[options.BooleanOption](),
		WAFMonitoringMode:                       options.Null[options.BooleanOption](),
		RedirectHTTPSToHTTP:                     options.Null[options.BooleanOption](),
		AntibotChallengeURLExceptions:           options.Null[options.ListOption](),
		AntibotFloodChallengeUserAgentWhitelist: options.Null[options.ListOption](),
	}

	blocks, err := flattenResourceOptions(&want)
	require.NoError(t, err)

	d := schema.TestResourceDataRaw(t, resourceWebsecResource().Schema, map[string]interface{}{})
	require.NoError(t, d.Set("options", blocks))

	require.JSONEq(t, sortedJSON(t, want), sortedJSON(t, expandResourceOptions(d, false)))
}

func TestEveryLocationOptionSurvivesTheSchemaRoundTrip(t *testing.T) {
	t.Parallel()

	want := locationOptionsWithEveryValue()

	blocks, err := flattenLocationOptions(&want)
	require.NoError(t, err)

	location := &schema.Resource{Schema: map[string]*schema.Schema{"options": locationOptionsSchema()}}
	d := schema.TestResourceDataRaw(t, location.Schema, map[string]interface{}{})
	require.NoError(t, d.Set("options", blocks))

	require.JSONEq(t, sortedJSON(t, want), sortedJSON(t, expandLocationOptions(d, false)))
}

func TestFlattenLeavesTheOptionsBlockOutWithoutAnOptionOfTheSchema(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceWebsecResource().Schema, map[string]interface{}{})
	createdWithoutOptions := expandResourceOptions(d, false)

	for name, opts := range map[string]*options.ResourceOptions{
		"no options object":    nil,
		"every option is null": createdWithoutOptions,
		"only unmanaged options set": {
			GeoACL:       options.Value(options.GeoACLOption{PolicyType: options.PolicyTypeDeny}),
			UseRSALECert: options.Value(options.BooleanOption{Enabled: true}),
		},
	} {
		blocks, err := flattenResourceOptions(opts)
		require.NoError(t, err, name)
		require.Equal(t, []interface{}{}, blocks, name)
	}

	blocks, err := flattenLocationOptions(nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{}, blocks)
}

func TestFlattenRefusesAnASNAboveTheInt32Range(t *testing.T) {
	t.Parallel()

	_, err := flattenResourceOptions(&options.ResourceOptions{
		LocationOptions: options.LocationOptions{
			ASNACL: options.Value(options.ASNACLOption{PolicyType: options.PolicyTypeDeny, ExceptedValues: []int64{64512, 4200000000}}),
		},
	})

	require.ErrorContains(t, err, "4200000000")
}

func TestNetworkHashTreatsEveryFormOfOneNetworkAsOneValue(t *testing.T) {
	t.Parallel()

	for _, same := range [][2]string{
		{"9.9.9.9", "9.9.9.9/32"},
		{"10.0.0.5/24", "10.0.0.0/24"},
		{"2001:DB8::1", "2001:db8::1/128"},
		{"2001:db8::1/64", "2001:db8::/64"},
	} {
		require.Equal(t, hashNetwork(same[0]), hashNetwork(same[1]), "%s and %s", same[0], same[1])
	}

	for _, different := range [][2]string{
		{"9.9.9.9", "9.9.9.8"},
		{"10.0.0.0/24", "10.0.0.0/25"},
	} {
		require.NotEqual(t, hashNetwork(different[0]), hashNetwork(different[1]), "%s and %s", different[0], different[1])
	}

	require.Equal(t, schema.HashString("not a network"), hashNetwork("not a network"))
}
