//go:build integration

package dns_test

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dnsmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dns/mock"
)

const (
	recordResourceName = "edgecenter_dns_zone_record"

	recordZone    = "example.com"
	recordDomain  = "www.example.com"
	recordTypeA   = "A"
	recordTTL     = 300
	recordContent = "1.2.3.4"

	recordStateID = `{"Zone":"example.com","Domain":"www.example.com","Type":"A"}`
)

func recordConfig(ttl int, records ...interface{}) map[string]interface{} {
	return map[string]interface{}{
		"zone":            recordZone,
		"domain":          recordDomain,
		"type":            recordTypeA,
		"ttl":             ttl,
		"meta":            []interface{}{map[string]interface{}{}},
		"resource_record": records,
	}
}

func recordPlain(content string) map[string]interface{} {
	return map[string]interface{}{"content": content, "enabled": true}
}

func recordWithMeta(content string, meta map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"content": content,
		"enabled": true,
		"meta":    []interface{}{meta},
	}
}

func recordFullMeta() map[string]interface{} {
	return map[string]interface{}{
		"ip":         []interface{}{"10.0.0.1"},
		"countries":  []interface{}{"us"},
		"continents": []interface{}{"na"},
		"notes":      []interface{}{"miami dc"},
		"asn":        []interface{}{12345},
		"regions":    []interface{}{"ru-spb"},
		"latlong":    []interface{}{27.98805612, 86.92527812},
		"default":    true,
		"backup":     true,
		"weight":     30,
	}
}

func recordAPIMeta() map[string]interface{} {
	return map[string]interface{}{
		"ip":        []interface{}{"10.0.0.9"},
		"countries": []interface{}{"de"},
		"asn":       []interface{}{float64(64512)},
		"latlong":   []interface{}{float64(50.0), float64(8.0)},
		"weight":    float64(70),
		"default":   true,
		"backup":    true,
	}
}

func recordRRSet(ttl int, contents ...string) dnssdk.RRSet {
	rrSet := dnssdk.RRSet{TTL: ttl}
	for _, content := range contents {
		rrSet.Records = append(rrSet.Records, *(&dnssdk.ResourceRecord{Enabled: true}).SetContent(recordTypeA, content))
	}

	return rrSet
}

func recordSingle(rrSet dnssdk.RRSet, ttl int, content string) bool {
	return rrSet.TTL == ttl &&
		len(rrSet.Records) == 1 &&
		rrSet.Records[0].ContentToString() == content &&
		rrSet.Records[0].Enabled
}

func recordAttrsWithSuffix(state *terraform.InstanceState, suffix string) []string {
	values := make([]string, 0)
	for key, value := range state.Attributes {
		if strings.HasSuffix(key, suffix) {
			values = append(values, value)
		}
	}
	sort.Strings(values)

	return values
}

func recordAttrKeysWithPrefix(state *terraform.InstanceState, prefix string) []string {
	keys := make([]string, 0)
	for key := range state.Attributes {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	return keys
}

func recordCreateCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return recordSingle(rrSet, recordTTL, recordContent) &&
				rrSet.Records[0].Meta == nil &&
				len(rrSet.Filters) == 0 &&
				rrSet.Meta != nil && rrSet.Meta.Failover == nil
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(900, "9.9.9.9"), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create builds the rrset and the chained read overwrites state",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
			support.RequireStateAttrs(t, state, map[string]string{
				"zone":              recordZone,
				"domain":            recordDomain,
				"type":              recordTypeA,
				"ttl":               "900",
				"resource_record.#": "1",
			})
			require.Equal(t, []string{"9.9.9.9"}, recordAttrsWithSuffix(state, ".content"))
		},
	}
}

func recordCreateTrimsCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	config := recordConfig(recordTTL, recordPlain(recordContent))
	config["zone"] = "  " + recordZone + "  "
	config["domain"] = " " + recordDomain + "\t"
	config["type"] = " " + recordTypeA + " "

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create trims whitespace around zone, domain and type",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
			support.RequireStateAttrs(t, state, map[string]string{
				"zone":   recordZone,
				"domain": recordDomain,
				"type":   recordTypeA,
			})
		},
	}
}

func recordCreateFiltersCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	wantFilters := []dnssdk.RecordFilter{
		{Type: "geodns", Limit: 2, Strict: true},
		{Type: "first_n", Limit: 1, Strict: false},
	}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return reflect.DeepEqual(rrSet.Filters, wantFilters)
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	config := recordConfig(recordTTL, recordPlain(recordContent))
	config["filter"] = []interface{}{
		map[string]interface{}{"type": "geodns", "limit": 2, "strict": true},
		map[string]interface{}{"type": "first_n", "limit": 1, "strict": false},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create sends filters in config order",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"filter.#":        "2",
				"filter.0.type":   "geodns",
				"filter.0.limit":  "2",
				"filter.0.strict": "true",
				"filter.1.type":   "first_n",
			})
		},
	}
}

func recordCreateManyRecordsCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	wantContents := map[string]bool{recordContent: true, "5.6.7.8": false}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			if len(rrSet.Records) != len(wantContents) {
				return false
			}

			for _, rec := range rrSet.Records {
				enabled, ok := wantContents[rec.ContentToString()]
				if !ok || enabled != rec.Enabled {
					return false
				}
			}

			return true
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	disabled := recordPlain("5.6.7.8")
	disabled["enabled"] = false

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create sends every record of the set with its enabled flag",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordPlain(recordContent), disabled),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"resource_record.#": "1"})
		},
	}
}

func recordCreateMetaWithoutRawConfigCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	wantMeta := map[string]interface{}{
		"ip":         []string{"10.0.0.1"},
		"countries":  []string{"us"},
		"continents": []string{"na"},
		"notes":      []string{"miami dc"},
		"asn":        []uint64{12345},
		"regions":    []string{"ru-spb"},
		"latlong":    []float64{27.988056, 86.925278},
	}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return len(rrSet.Records) == 1 && reflect.DeepEqual(rrSet.Records[0].Meta, wantMeta)
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "default, backup and weight are dropped when the raw config is unavailable",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordWithMeta(recordContent, recordFullMeta())),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordCreateFailoverCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	failover := dnssdk.FailoverMeta{
		Protocol:       "HTTP",
		Port:           8080,
		Frequency:      30,
		Timeout:        10,
		Method:         "GET",
		Url:            "/health",
		Tls:            true,
		Regexp:         "ok",
		HTTPStatusCode: 200,
		Host:           "probe.example.com",
		Verify:         true,
	}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return rrSet.Meta != nil && rrSet.Meta.Failover != nil && *rrSet.Meta.Failover == failover
		}),
	).Return(nil)

	result := recordRRSet(recordTTL, recordContent)
	result.Meta = &dnssdk.Meta{Failover: &failover}
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	config := recordConfig(recordTTL, recordPlain(recordContent))
	config["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":         failover.Protocol,
					"port":             failover.Port,
					"frequency":        failover.Frequency,
					"timeout":          failover.Timeout,
					"method":           failover.Method,
					"url":              failover.Url,
					"tls":              failover.Tls,
					"regexp":           failover.Regexp,
					"http_status_code": failover.HTTPStatusCode,
					"host":             failover.Host,
					"verify":           failover.Verify,
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create sends the http failover meta",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"meta.0.failover.#":                  "1",
				"meta.0.failover.0.protocol":         "HTTP",
				"meta.0.failover.0.url":              "/health",
				"meta.0.failover.0.http_status_code": "200",
				"meta.0.failover.0.tls":              "true",
				"meta.0.failover.0.verify":           "true",
			})
		},
	}
}

func recordCreateFailoverRejectedCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	config := recordConfig(recordTTL, recordPlain(recordContent))
	config["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":  "TCP",
					"port":      53,
					"frequency": 30,
					"timeout":   10,
					"url":       "/health",
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "failover url without the http protocol is rejected before any api call",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "failover URL can only be set along with HTTP protocol")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func recordCreateZoneFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, recordZone).
		Return(dnssdk.Zone{}, fmt.Errorf("api error: zone not found"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "api error while checking that the zone exists",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "find zone: api error: zone not found")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func recordCreateRRSetFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).
		Return(fmt.Errorf("api error: rrset already exists"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "api error on create",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create zone rrset: api error: rrset already exists")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func recordCreateReadFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(dnssdk.RRSet{}, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "read failure after create leaves the bare zone as id",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get zone rrset: api error: server unavailable")
			support.RequireStateID(t, state, recordZone)
		},
	}
}

func recordReadCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(1200, "8.8.8.8", "8.8.4.4"), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read replaces ttl and records and rewrites the id as json",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
			support.RequireStateAttrs(t, state, map[string]string{
				"ttl":               "1200",
				"resource_record.#": "2",
			})
			require.Equal(t, []string{"8.8.4.4", "8.8.8.8"}, recordAttrsWithSuffix(state, ".content"))
		},
	}
}

func recordReadKeepsFilterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	current["filter"] = []interface{}{
		map[string]interface{}{"type": "geodns", "limit": 4, "strict": true},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read returning zero filters keeps the filter already in state",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"filter.#":       "1",
				"filter.0.type":  "geodns",
				"filter.0.limit": "4",
			})
		},
	}
}

func recordReadKeepsRecordsCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(dnssdk.RRSet{}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read returning no records zeroes the ttl but keeps the records in state",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"ttl":               "0",
				"resource_record.#": "1",
			})
			require.Equal(t, []string{recordContent}, recordAttrsWithSuffix(state, ".content"))
		},
	}
}

func recordReadFiltersCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	result := recordRRSet(recordTTL, recordContent)
	result.Filters = []dnssdk.RecordFilter{{Type: "geodistance", Limit: 7, Strict: false}}
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	current["filter"] = []interface{}{
		map[string]interface{}{"type": "geodns", "limit": 4, "strict": true},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read overwrites the filter with the api one",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"filter.#":        "1",
				"filter.0.type":   "geodistance",
				"filter.0.limit":  "7",
				"filter.0.strict": "false",
			})
		},
	}
}

func recordReadRecordMetaCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	result := recordRRSet(recordTTL, "7.7.7.7")
	result.Records[0].Meta = recordAPIMeta()
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read maps record meta back into state",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: recordConfig(recordTTL, recordWithMeta(recordContent, recordFullMeta())),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, []string{"7.7.7.7"}, recordAttrsWithSuffix(state, ".content"))
			require.Equal(t, []string{"10.0.0.9"}, recordAttrsWithSuffix(state, ".ip.0"))
			require.Equal(t, []string{"de"}, recordAttrsWithSuffix(state, ".countries.0"))
			require.Equal(t, []string{"64512"}, recordAttrsWithSuffix(state, ".asn.0"))
			require.Equal(t, []string{"70"}, recordAttrsWithSuffix(state, ".weight"))
			require.Equal(t, []string{"true"}, recordAttrsWithSuffix(state, ".backup"))
		},
	}
}

func recordReadFailoverCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	result := recordRRSet(recordTTL, recordContent)
	result.Meta = &dnssdk.Meta{Failover: &dnssdk.FailoverMeta{
		Protocol:  "TCP",
		Port:      53,
		Frequency: 15,
		Timeout:   5,
		Tls:       false,
		Verify:    true,
	}}
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	current["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":  "HTTP",
					"port":      80,
					"frequency": 30,
					"timeout":   10,
					"verify":    true,
					"tls":       true,
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read drops verify when tls is off",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"meta.0.failover.0.protocol":  "TCP",
				"meta.0.failover.0.port":      "53",
				"meta.0.failover.0.frequency": "15",
				"meta.0.failover.0.timeout":   "5",
				"meta.0.failover.0.tls":       "false",
				"meta.0.failover.0.verify":    "false",
			})
		},
	}
}

func recordReadEmptyFailoverCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	result := recordRRSet(recordTTL, recordContent)
	result.Meta = &dnssdk.Meta{}
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	current["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":  "HTTP",
					"port":      80,
					"frequency": 30,
					"timeout":   10,
					"url":       "/health",
					"tls":       true,
					"verify":    true,
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read of a meta without failover wipes the failover already in state",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"meta.#": "1"})
			require.Empty(t, recordAttrKeysWithPrefix(state, "meta.0."),
				"an empty api meta must leave nothing of the configured failover block")
		},
	}
}

func recordReadAssignedTTLCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(300, recordContent), nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	delete(current, "ttl")

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read stores the ttl the api assigned when the config declares none",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"ttl": "300"})
		},
	}
}

func recordReadTrailingDotCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	const (
		cnameType   = "CNAME"
		cnameBare   = "target.example.com"
		cnameDotted = "target.example.com."
	)

	result := dnssdk.RRSet{TTL: recordTTL}
	result.Records = append(result.Records,
		*(&dnssdk.ResourceRecord{Enabled: true}).SetContent(cnameType, cnameDotted))
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, cnameType).Return(result, nil)

	current := recordConfig(recordTTL, recordPlain(cnameBare))
	current["type"] = cnameType

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read keeps the trailing dot the api returns for cname content",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{"resource_record.#": "1"})
			require.Equal(t, []string{cnameDotted}, recordAttrsWithSuffix(state, ".content"))
		},
	}
}

func recordReadHealthFilterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	result := recordRRSet(recordTTL, recordContent)
	result.Filters = []dnssdk.RecordFilter{{Type: "is_healthy", Limit: 0, Strict: false}}
	result.Meta = &dnssdk.Meta{Failover: &dnssdk.FailoverMeta{
		Protocol:  "TCP",
		Port:      53,
		Frequency: 15,
		Timeout:   5,
	}}
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(result, nil)

	current := recordConfig(recordTTL, recordPlain(recordContent))
	current["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":  "TCP",
					"port":      53,
					"frequency": 15,
					"timeout":   5,
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read adds the health filter the api attached to a failover record",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordZone,
		CurrentState: current,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"filter.#":        "1",
				"filter.0.type":   "is_healthy",
				"filter.0.limit":  "0",
				"filter.0.strict": "false",
			})
		},
	}
}

func recordReadNotFoundCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(dnssdk.RRSet{}, dnssdk.APIError{StatusCode: http.StatusNotFound, Message: "rrset not found"})

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read of a missing rrset reports an error and keeps the id",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get zone rrset: 404: rrset not found")
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordReadFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(dnssdk.RRSet{}, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "api error on read",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get zone rrset: api error: server unavailable")
		},
	}
}

func recordUpdateCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return recordSingle(rrSet, 600, "5.6.7.8")
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(600, "5.6.7.8"), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "update sends the new ttl and content",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		NewConfig:    recordConfig(600, recordPlain("5.6.7.8")),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
			support.RequireStateAttrs(t, state, map[string]string{
				"ttl":               "600",
				"resource_record.#": "1",
			})
			require.Equal(t, []string{"5.6.7.8"}, recordAttrsWithSuffix(state, ".content"))
		},
	}
}

func recordUpdateWithoutTTLCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).
		Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(0, recordContent), nil)

	config := recordConfig(recordTTL, recordPlain(recordContent))
	delete(config, "ttl")

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "dropping ttl from the config sends a zero ttl to the api",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			fake.Client.AssertCalled(t, "UpdateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
				mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
					return recordSingle(rrSet, 0, recordContent)
				}))
			support.RequireStateAttrs(t, state, map[string]string{"ttl": "0"})
		},
	}
}

func recordUpdateFailoverRejectedCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).
		Return(nil).Maybe()
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil).Maybe()

	config := recordConfig(600, recordPlain("5.6.7.8"))
	config["meta"] = []interface{}{
		map[string]interface{}{
			"failover": []interface{}{
				map[string]interface{}{
					"protocol":  "UDP",
					"port":      53,
					"frequency": 30,
					"timeout":   10,
					"host":      "probe.example.com",
				},
			},
		},
	}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "failover host without the http protocol is rejected before the update call",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		NewConfig:    config,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "failover host can only be set along with HTTP protocol")
			support.RequireStateID(t, state, recordStateID)
			fake.Client.AssertNotCalled(t, "UpdateRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			fake.Client.AssertNotCalled(t, "RRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		},
	}
}

func recordUpdateFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("UpdateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA, mock.Anything).
		Return(fmt.Errorf("api error: rrset is locked"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "api error on update keeps the previous id",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		NewConfig:    recordConfig(600, recordPlain("5.6.7.8")),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "update zone rrset: api error: rrset is locked")
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordDeleteCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteRRSet", mock.Anything, recordZone, recordDomain, recordTypeA).Return(nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "delete clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func recordDeleteFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteRRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(fmt.Errorf("api error: rrset is in use"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "api error on delete keeps the id",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    recordStateID,
		CurrentState: recordConfig(recordTTL, recordPlain(recordContent)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "delete zone rrset: api error: rrset is in use")
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordCreateFullMetaCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	wantMeta := map[string]interface{}{
		"ip":         []string{"10.0.0.1"},
		"countries":  []string{"us"},
		"continents": []string{"na"},
		"notes":      []string{"miami dc"},
		"asn":        []uint64{12345},
		"regions":    []string{"ru-spb"},
		"latlong":    []float64{27.988056, 86.925278},
		"default":    true,
		"backup":     true,
		"weight":     30,
	}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return len(rrSet.Records) == 1 && reflect.DeepEqual(rrSet.Records[0].Meta, wantMeta)
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create sends every meta field and truncates latlong to six decimals",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordWithMeta(recordContent, recordFullMeta())),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordCreateFalsyMetaCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	wantMeta := map[string]interface{}{
		"default": false,
		"backup":  false,
		"weight":  0,
	}

	mc.Client.On("Zone", mock.Anything, recordZone).Return(dnssdk.Zone{Name: recordZone}, nil)
	mc.Client.On("CreateRRSet", mock.Anything, recordZone, recordDomain, recordTypeA,
		mock.MatchedBy(func(rrSet dnssdk.RRSet) bool {
			return len(rrSet.Records) == 1 && reflect.DeepEqual(rrSet.Records[0].Meta, wantMeta)
		}),
	).Return(nil)
	mc.Client.On("RRSet", mock.Anything, recordZone, recordDomain, recordTypeA).
		Return(recordRRSet(recordTTL, recordContent), nil)

	meta := map[string]interface{}{"default": false, "backup": false, "weight": 0}

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "explicit false and zero meta values are still sent",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: recordConfig(recordTTL, recordWithMeta(recordContent, meta)),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, recordStateID)
		},
	}
}

func recordCtyValue(value interface{}) cty.Value {
	switch v := value.(type) {
	case string:
		return cty.StringVal(v)
	case bool:
		return cty.BoolVal(v)
	case int:
		return cty.NumberIntVal(int64(v))
	case float64:
		return cty.NumberFloatVal(v)
	case []interface{}:
		if len(v) == 0 {
			return cty.EmptyTupleVal
		}

		items := make([]cty.Value, len(v))
		for i, item := range v {
			items[i] = recordCtyValue(item)
		}

		return cty.TupleVal(items)
	case map[string]interface{}:
		if len(v) == 0 {
			return cty.EmptyObjectVal
		}

		attrs := make(map[string]cty.Value, len(v))
		for key, item := range v {
			attrs[key] = recordCtyValue(item)
		}

		return cty.ObjectVal(attrs)
	default:
		return cty.NullVal(cty.DynamicPseudoType)
	}
}

func recordRawConfigRunner(
	t *testing.T,
	res *schema.Resource,
	tc support.ResourceCase[*dnsmock.MockedDNS],
	fake *dnsmock.MockedDNS,
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	ctx := context.Background()

	diff, err := res.Diff(ctx, nil, terraform.NewResourceConfigRaw(tc.NewConfig), fake.TestMeta())
	require.NoError(t, err)

	diff.RawConfig = recordCtyValue(tc.NewConfig)

	return res.Apply(ctx, nil, diff, fake.TestMeta())
}

func TestIntegrationZoneRecord_TableDriven(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, recordResourceName)

	cases := []support.ResourceCase[*dnsmock.MockedDNS]{
		recordCreateCase(),
		recordCreateTrimsCase(),
		recordCreateFiltersCase(),
		recordCreateManyRecordsCase(),
		recordCreateMetaWithoutRawConfigCase(),
		recordCreateFailoverCase(),
		recordCreateFailoverRejectedCase(),
		recordCreateZoneFailureCase(),
		recordCreateRRSetFailureCase(),
		recordCreateReadFailureCase(),
		recordReadCase(),
		recordReadKeepsFilterCase(),
		recordReadKeepsRecordsCase(),
		recordReadFiltersCase(),
		recordReadRecordMetaCase(),
		recordReadFailoverCase(),
		recordReadEmptyFailoverCase(),
		recordReadAssignedTTLCase(),
		recordReadTrailingDotCase(),
		recordReadHealthFilterCase(),
		recordReadNotFoundCase(),
		recordReadFailureCase(),
		recordUpdateCase(),
		recordUpdateWithoutTTLCase(),
		recordUpdateFailoverRejectedCase(),
		recordUpdateFailureCase(),
		recordDeleteCase(),
		recordDeleteFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*dnsmock.MockedDNS])
}

func TestIntegrationZoneRecordRawConfigMeta(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, recordResourceName)

	cases := []support.ResourceCase[*dnsmock.MockedDNS]{
		recordCreateFullMetaCase(),
		recordCreateFalsyMetaCase(),
	}

	support.RunResourceCases(t, resource, cases, recordRawConfigRunner)
}

func recordEmptyIDMock() *dnsmock.MockedDNS {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("RRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(recordRRSet(recordTTL, recordContent), nil).Maybe()
	mc.Client.On("UpdateRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()
	mc.Client.On("DeleteRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	return mc
}

func TestIntegrationZoneRecordEmptyID(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, recordResourceName)

	cases := []struct {
		name string
		run  func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics
	}{
		{name: "read", run: resource.ReadContext},
		{name: "update", run: resource.UpdateContext},
		{name: "delete", run: resource.DeleteContext},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := recordEmptyIDMock()
			t.Cleanup(func() { mc.MockCleanup(t) })

			data := resource.Data(nil)
			diags := tt.run(context.Background(), data, mc.Config)

			support.RequireOnlyErrorDiags(t, diags)
			require.Len(t, diags, 1)
			support.RequireErrorDiagContains(t, diags, "empty id")
			require.Empty(t, data.Id(), "the guarded call must not touch the id")
			mc.Client.AssertNotCalled(t, "RRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mc.Client.AssertNotCalled(t, "UpdateRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mc.Client.AssertNotCalled(t, "DeleteRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
