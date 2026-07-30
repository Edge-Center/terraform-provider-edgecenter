package dns

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	ctyjson "github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"
)

func mapperData(t *testing.T, records, filters []interface{}) *schema.ResourceData {
	t.Helper()

	res := resourceDNSZoneRecord()
	raw, err := json.Marshal(map[string]interface{}{
		"id":                              "example.com",
		DNSZoneRecordSchemaZone:           "example.com",
		DNSZoneRecordSchemaDomain:         "www.example.com",
		DNSZoneRecordSchemaType:           "A",
		DNSZoneRecordSchemaTTL:            60,
		DNSZoneRecordSchemaResourceRecord: records,
		DNSZoneRecordSchemaFilter:         filters,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	cfg, err := ctyjson.Unmarshal(raw, res.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	state, err := res.ShimInstanceStateFromValue(cfg)
	if err != nil {
		t.Fatalf("shim state: %v", err)
	}
	state.RawConfig = cfg

	return res.Data(state)
}

func mapperRecord(content string, enabled bool, meta map[string]interface{}) map[string]interface{} {
	record := map[string]interface{}{
		DNSZoneRecordSchemaContent: content,
		DNSZoneRecordSchemaEnabled: enabled,
	}
	if meta != nil {
		record[DNSZoneRecordSchemaMeta] = []interface{}{meta}
	}

	return record
}

func mapperFilter(fType string, limit, strict interface{}) map[string]interface{} {
	return map[string]interface{}{
		DNSZoneRecordSchemaFilterType:   fType,
		DNSZoneRecordSchemaFilterLimit:  limit,
		DNSZoneRecordSchemaFilterStrict: strict,
	}
}

func mapperFillOneRecord(t *testing.T, rType string, record map[string]interface{}) dnssdk.ResourceRecord {
	t.Helper()

	rrSet := dnssdk.RRSet{Records: make([]dnssdk.ResourceRecord, 0)}
	if err := fillRRSet(mapperData(t, []interface{}{record}, nil), rType, &rrSet); err != nil {
		t.Fatalf("fillRRSet: %v", err)
	}
	if len(rrSet.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rrSet.Records))
	}

	return rrSet.Records[0]
}

func mapperFullFailover() *dnssdk.FailoverMeta {
	return &dnssdk.FailoverMeta{
		Protocol:       "HTTP",
		Port:           443,
		Frequency:      30,
		Timeout:        5,
		Method:         "GET",
		Url:            "/health",
		Tls:            true,
		Regexp:         "^ok$",
		HTTPStatusCode: 200,
		Host:           "probe.example.com",
		Verify:         true,
	}
}

func TestRecordSetContentTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rType       string
		content     string
		wantContent []interface{}
		wantString  string
	}{
		{
			name:        "a CAA value containing a space gets a trailing empty slot",
			rType:       "CAA",
			content:     `0 issue "company.org; account=12345"`,
			wantContent: []interface{}{int64(0), "issue", `"company.org; account=12345"`, nil},
			wantString:  `0 issue "company.org; account=12345" <nil>`,
		},
		{
			name:        "a CAA value of three tokens round trips",
			rType:       "CAA",
			content:     `0 issue "company.org"`,
			wantContent: []interface{}{int64(0), "issue", `"company.org"`},
			wantString:  `0 issue "company.org"`,
		},
		{
			name:    "MX without a priority yields empty content",
			rType:   "MX",
			content: "mail.example.com.",
		},
		{
			name:    "MX with a double space yields empty content",
			rType:   "MX",
			content: "10  mail.example.com.",
		},
		{
			name:    "SRV with three tokens yields empty content",
			rType:   "SRV",
			content: "10 20 5060",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := (&dnssdk.ResourceRecord{}).SetContent(tt.rType, tt.content)
			if len(got.Content) != len(tt.wantContent) {
				t.Fatalf("content length = %d, want %d: %#v", len(got.Content), len(tt.wantContent), got.Content)
			}
			if !reflect.DeepEqual(got.Content, tt.wantContent) {
				t.Errorf("content:\n got: %#v\nwant: %#v", got.Content, tt.wantContent)
			}
			if got.ContentToString() != tt.wantString {
				t.Errorf("content string:\n got: %q\nwant: %q", got.ContentToString(), tt.wantString)
			}
		})
	}
}

func TestRecordFillRRSetMetaKinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		meta map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "ip",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaIP: []interface{}{"10.0.0.1", "10.0.0.2"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaIP: []string{"10.0.0.1", "10.0.0.2"}},
		},
		{
			name: "countries",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaCountries: []interface{}{"nl", "de"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaCountries: []string{"nl", "de"}},
		},
		{
			name: "continents",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaContinents: []interface{}{"eu", "as"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaContinents: []string{"eu", "as"}},
		},
		{
			name: "notes",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaNotes: []interface{}{"Miami DC", "backup DC"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaNotes: []string{"Miami DC", "backup DC"}},
		},
		{
			name: "asn",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaAsn: []interface{}{12345, 65000}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaAsn: []uint64{12345, 65000}},
		},
		{
			name: "regions",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaRegions: []interface{}{"ru-spb", "ru-mow"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaRegions: []string{"ru-mow", "ru-spb"}},
		},
		{
			name: "latlong",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaLatLong: []interface{}{27.988056, 86.925278}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaLatLong: []float64{27.988056, 86.925278}},
		},
		{
			name: "latlong is stored with six decimal places",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaLatLong: []interface{}{51.50735820, -0.12775820}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaLatLong: []float64{51.507358, -0.127758}},
		},
		{
			name: "default true",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaDefault: true},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaDefault: true},
		},
		{
			name: "default false is still sent when written explicitly",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaDefault: false},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaDefault: false},
		},
		{
			name: "backup true",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaBackup: true},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaBackup: true},
		},
		{
			name: "backup false is still sent when written explicitly",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaBackup: false},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaBackup: false},
		},
		{
			name: "weight",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55},
		},
		{
			name: "weight zero is still sent when written explicitly",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 0},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 0},
		},
		{
			name: "omitted backup, weight and default are not sent",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaNotes: []interface{}{"only notes"}},
			want: map[string]interface{}{DNSZoneRecordSchemaMetaNotes: []string{"only notes"}},
		},
		{
			name: "empty lists are not sent",
			meta: map[string]interface{}{
				DNSZoneRecordSchemaMetaIP:         []interface{}{},
				DNSZoneRecordSchemaMetaCountries:  []interface{}{},
				DNSZoneRecordSchemaMetaContinents: []interface{}{},
				DNSZoneRecordSchemaMetaNotes:      []interface{}{},
				DNSZoneRecordSchemaMetaAsn:        []interface{}{},
				DNSZoneRecordSchemaMetaRegions:    []interface{}{},
				DNSZoneRecordSchemaMetaLatLong:    []interface{}{},
			},
			want: nil,
		},
		{
			name: "latlong with a single coordinate is skipped",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaLatLong: []interface{}{27.988056}},
			want: nil,
		},
		{
			name: "invalid ip is dropped without an error",
			meta: map[string]interface{}{DNSZoneRecordSchemaMetaIP: []interface{}{"10.0.0.1", "not-an-ip"}},
			want: nil,
		},
		{
			name: "every kind at once",
			meta: map[string]interface{}{
				DNSZoneRecordSchemaMetaIP:         []interface{}{"10.0.0.1"},
				DNSZoneRecordSchemaMetaCountries:  []interface{}{"nl"},
				DNSZoneRecordSchemaMetaContinents: []interface{}{"eu"},
				DNSZoneRecordSchemaMetaNotes:      []interface{}{"Miami DC"},
				DNSZoneRecordSchemaMetaAsn:        []interface{}{12345},
				DNSZoneRecordSchemaMetaRegions:    []interface{}{"ru-spb"},
				DNSZoneRecordSchemaMetaLatLong:    []interface{}{27.988056, 86.925278},
				DNSZoneRecordSchemaMetaBackup:     true,
				DNSZoneRecordSchemaMetaWeight:     10,
				DNSZoneRecordSchemaMetaDefault:    true,
			},
			want: map[string]interface{}{
				DNSZoneRecordSchemaMetaIP:         []string{"10.0.0.1"},
				DNSZoneRecordSchemaMetaCountries:  []string{"nl"},
				DNSZoneRecordSchemaMetaContinents: []string{"eu"},
				DNSZoneRecordSchemaMetaNotes:      []string{"Miami DC"},
				DNSZoneRecordSchemaMetaAsn:        []uint64{12345},
				DNSZoneRecordSchemaMetaRegions:    []string{"ru-spb"},
				DNSZoneRecordSchemaMetaLatLong:    []float64{27.988056, 86.925278},
				DNSZoneRecordSchemaMetaBackup:     true,
				DNSZoneRecordSchemaMetaWeight:     10,
				DNSZoneRecordSchemaMetaDefault:    true,
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapperFillOneRecord(t, "A", mapperRecord("1.2.3.4", true, tt.meta)).Meta
			if regions, ok := got[DNSZoneRecordSchemaMetaRegions].([]string); ok {
				sort.Strings(regions)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("meta:\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestRecordFillRRSetRecordFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rType       string
		content     string
		enabled     bool
		meta        map[string]interface{}
		wantContent []interface{}
		wantMeta    map[string]interface{}
	}{
		{
			name:        "A record without a meta block",
			rType:       "A",
			content:     "1.2.3.4",
			enabled:     true,
			wantContent: []interface{}{"1.2.3.4"},
		},
		{
			name:        "disabled record",
			rType:       "A",
			content:     "1.2.3.4",
			enabled:     false,
			wantContent: []interface{}{"1.2.3.4"},
		},
		{
			name:        "MX content is split into priority and target",
			rType:       "MX",
			content:     "50 mail.company.io.",
			enabled:     true,
			wantContent: []interface{}{int64(50), "mail.company.io."},
		},
		{
			name:        "MX priority that is not a number becomes zero",
			rType:       "MX",
			content:     "abc mail.example.com.",
			enabled:     true,
			wantContent: []interface{}{int64(0), "mail.example.com."},
		},
		{
			name:    "MX without a priority is kept with empty content",
			rType:   "MX",
			content: "mail.example.com.",
			enabled: true,
		},
		{
			name:    "MX with a double space is kept with empty content",
			rType:   "MX",
			content: "10  mail.example.com.",
			enabled: true,
		},
		{
			name:    "SRV with three tokens is kept with empty content",
			rType:   "SRV",
			content: "10 20 5060",
			enabled: true,
		},
		{
			name:        "empty meta block adds no meta",
			rType:       "A",
			content:     "1.2.3.4",
			enabled:     true,
			meta:        map[string]interface{}{},
			wantContent: []interface{}{"1.2.3.4"},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapperFillOneRecord(t, tt.rType, mapperRecord(tt.content, tt.enabled, tt.meta))
			if !reflect.DeepEqual(got.Content, tt.wantContent) {
				t.Errorf("content:\n got: %#v\nwant: %#v", got.Content, tt.wantContent)
			}
			if got.Enabled != tt.enabled {
				t.Errorf("enabled = %v, want %v", got.Enabled, tt.enabled)
			}
			if !reflect.DeepEqual(got.Meta, tt.wantMeta) {
				t.Errorf("meta:\n got: %#v\nwant: %#v", got.Meta, tt.wantMeta)
			}
		})
	}
}

func TestRecordFillRRSetKeepsEveryRecord(t *testing.T) {
	t.Parallel()

	records := []interface{}{
		mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 10}),
		mapperRecord("5.6.7.8", false, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 90}),
	}

	rrSet := dnssdk.RRSet{Records: make([]dnssdk.ResourceRecord, 0)}
	if err := fillRRSet(mapperData(t, records, nil), "A", &rrSet); err != nil {
		t.Fatalf("fillRRSet: %v", err)
	}

	want := map[string]int{"1.2.3.4": 10, "5.6.7.8": 90}
	got := map[string]int{}
	for _, rec := range rrSet.Records {
		got[rec.ContentToString()] = rec.Meta[DNSZoneRecordSchemaMetaWeight].(int)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("records:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestRecordFillRRSetFilters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		filters []interface{}
		want    []dnssdk.RecordFilter
	}{
		{
			name: "no filter block leaves the filter slice nil",
		},
		{
			name:    "full filter",
			filters: []interface{}{mapperFilter("geodns", 3, true)},
			want:    []dnssdk.RecordFilter{{Type: "geodns", Limit: 3, Strict: true}},
		},
		{
			name:    "omitted limit and strict become their zero values",
			filters: []interface{}{mapperFilter("first_n", nil, nil)},
			want:    []dnssdk.RecordFilter{{Type: "first_n"}},
		},
		{
			name: "order is preserved",
			filters: []interface{}{
				mapperFilter("geodistance", 1, false),
				mapperFilter("is_healthy", 2, true),
				mapperFilter("default", 0, false),
			},
			want: []dnssdk.RecordFilter{
				{Type: "geodistance", Limit: 1},
				{Type: "is_healthy", Limit: 2, Strict: true},
				{Type: "default"},
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			records := []interface{}{mapperRecord("1.2.3.4", true, nil)}
			rrSet := dnssdk.RRSet{Records: make([]dnssdk.ResourceRecord, 0)}
			if err := fillRRSet(mapperData(t, records, tt.filters), "A", &rrSet); err != nil {
				t.Fatalf("fillRRSet: %v", err)
			}
			if !reflect.DeepEqual(rrSet.Filters, tt.want) {
				t.Fatalf("filters:\n got: %#v\nwant: %#v", rrSet.Filters, tt.want)
			}
		})
	}
}

func TestRecordFillRRSetInvalidMetaErrorIsUnreachable(t *testing.T) {
	t.Parallel()

	if dnssdk.NewResourceMetaLatLong("27.988056").Valid() == nil {
		t.Fatal("the SDK no longer rejects a malformed latlong, the error path may now be reachable")
	}

	cases := []struct {
		name     string
		lat      float64
		long     float64
		wantMeta []float64
	}{
		{name: "positive", lat: 27.988056, long: 86.925278, wantMeta: []float64{27.988056, 86.925278}},
		{name: "negative", lat: -33.865143, long: -151.209900, wantMeta: []float64{-33.865143, -151.2099}},
		{name: "zero", lat: 0, long: 0, wantMeta: []float64{0, 0}},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta := map[string]interface{}{
				DNSZoneRecordSchemaMetaLatLong: []interface{}{tt.lat, tt.long},
				DNSZoneRecordSchemaMetaDefault: true,
			}
			got := mapperFillOneRecord(t, "A", mapperRecord("1.2.3.4", true, meta)).Meta
			if !reflect.DeepEqual(got[DNSZoneRecordSchemaMetaLatLong], tt.wantMeta) {
				t.Fatalf("latlong:\n got: %#v\nwant: %#v", got[DNSZoneRecordSchemaMetaLatLong], tt.wantMeta)
			}
		})
	}
}

func TestRecordHasExplicitResourceRecordMetaField(t *testing.T) {
	t.Parallel()

	t.Run("null raw config", func(t *testing.T) {
		t.Parallel()

		d := schema.TestResourceDataRaw(t, resourceDNSZoneRecord().Schema, map[string]interface{}{
			DNSZoneRecordSchemaZone: "example.com",
			DNSZoneRecordSchemaResourceRecord: []interface{}{
				map[string]interface{}{
					DNSZoneRecordSchemaContent: "1.2.3.4",
					DNSZoneRecordSchemaMeta: []interface{}{
						map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55},
					},
				},
			},
		})
		if !d.GetRawConfig().IsNull() {
			t.Fatal("expected TestResourceDataRaw to leave the raw config null")
		}
		if hasExplicitResourceRecordMetaField(d, "1.2.3.4", DNSZoneRecordSchemaMetaWeight) {
			t.Fatal("expected false when the raw config is null")
		}
	})

	cases := []struct {
		name    string
		records []interface{}
		content string
		field   string
		want    bool
	}{
		{
			name:    "field written",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55})},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    true,
		},
		{
			name:    "field written as a zero value",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 0})},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    true,
		},
		{
			name:    "field written as false",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaBackup: false})},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaBackup,
			want:    true,
		},
		{
			name:    "another field written",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaBackup: true})},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
		{
			name:    "no meta block",
			records: []interface{}{mapperRecord("1.2.3.4", true, nil)},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
		{
			name:    "empty meta block",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{})},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
		{
			name:    "unknown content",
			records: []interface{}{mapperRecord("1.2.3.4", true, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55})},
			content: "9.9.9.9",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
		{
			name: "second record is found",
			records: []interface{}{
				mapperRecord("1.2.3.4", true, nil),
				mapperRecord("5.6.7.8", true, map[string]interface{}{DNSZoneRecordSchemaMetaWeight: 55}),
			},
			content: "5.6.7.8",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    true,
		},
		{
			name:    "no records at all",
			records: []interface{}{},
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
		{
			name:    "null resource_record",
			records: nil,
			content: "1.2.3.4",
			field:   DNSZoneRecordSchemaMetaWeight,
			want:    false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hasExplicitResourceRecordMetaField(mapperData(t, tt.records, nil), tt.content, tt.field)
			if got != tt.want {
				t.Fatalf("hasExplicitResourceRecordMetaField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecordListToFailoverMeta(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		list []interface{}
		want *dnssdk.FailoverMeta
	}{
		{
			name: "empty list",
			list: []interface{}{},
		},
		{
			name: "nil first element",
			list: []interface{}{nil},
		},
		{
			name: "no failover key",
			list: []interface{}{map[string]interface{}{}},
		},
		{
			name: "empty failover block",
			list: []interface{}{map[string]interface{}{DNSZoneRecordSchemaFailover: []interface{}{}}},
		},
		{
			name: "full failover block",
			list: []interface{}{map[string]interface{}{
				DNSZoneRecordSchemaFailover: []interface{}{map[string]interface{}{
					DNSZoneRecordSchemaFailoverProtocol:       "HTTP",
					DNSZoneRecordSchemaFailoverPort:           443,
					DNSZoneRecordSchemaFailoverFrequency:      30,
					DNSZoneRecordSchemaFailoverTimeout:        5,
					DNSZoneRecordSchemaFailoverMethod:         "GET",
					DNSZoneRecordSchemaFailoverURL:            "/health",
					DNSZoneRecordSchemaFailoverTLS:            true,
					DNSZoneRecordSchemaFailoverRegexp:         "^ok$",
					DNSZoneRecordSchemaFailoverHTTPStatusCode: 200,
					DNSZoneRecordSchemaFailoverHost:           "probe.example.com",
					DNSZoneRecordSchemaFailoverVerify:         true,
				}},
			}},
			want: mapperFullFailover(),
		},
		{
			name: "only the required fields",
			list: []interface{}{map[string]interface{}{
				DNSZoneRecordSchemaFailover: []interface{}{map[string]interface{}{
					DNSZoneRecordSchemaFailoverProtocol:  "ICMP",
					DNSZoneRecordSchemaFailoverPort:      0,
					DNSZoneRecordSchemaFailoverFrequency: 10,
					DNSZoneRecordSchemaFailoverTimeout:   1,
				}},
			}},
			want: &dnssdk.FailoverMeta{Protocol: "ICMP", Frequency: 10, Timeout: 1},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := listToFailoverMeta(tt.list)
			if !reflect.DeepEqual(got.Failover, tt.want) {
				t.Fatalf("failover:\n got: %#v\nwant: %#v", got.Failover, tt.want)
			}
		})
	}
}

func TestRecordListToFailoverMetaReadsSchemaData(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceDNSZoneRecord().Schema, map[string]interface{}{
		DNSZoneRecordSchemaRRSetMeta: []interface{}{
			map[string]interface{}{
				DNSZoneRecordSchemaFailover: []interface{}{
					map[string]interface{}{
						DNSZoneRecordSchemaFailoverProtocol:       "HTTP",
						DNSZoneRecordSchemaFailoverPort:           443,
						DNSZoneRecordSchemaFailoverFrequency:      30,
						DNSZoneRecordSchemaFailoverTimeout:        5,
						DNSZoneRecordSchemaFailoverMethod:         "GET",
						DNSZoneRecordSchemaFailoverURL:            "/health",
						DNSZoneRecordSchemaFailoverTLS:            true,
						DNSZoneRecordSchemaFailoverRegexp:         "^ok$",
						DNSZoneRecordSchemaFailoverHTTPStatusCode: 200,
						DNSZoneRecordSchemaFailoverHost:           "probe.example.com",
						DNSZoneRecordSchemaFailoverVerify:         true,
					},
				},
			},
		},
	})

	got := listToFailoverMeta(d.Get(DNSZoneRecordSchemaRRSetMeta).([]interface{}))
	if !reflect.DeepEqual(got.Failover, mapperFullFailover()) {
		t.Fatalf("failover:\n got: %#v\nwant: %#v", got.Failover, mapperFullFailover())
	}
}

func TestRecordFailoverMetaToList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		failover *dnssdk.FailoverMeta
		want     map[string]interface{}
	}{
		{
			name: "nil failover still produces one element",
		},
		{
			name:     "zero failover produces an empty block",
			failover: &dnssdk.FailoverMeta{},
			want:     map[string]interface{}{},
		},
		{
			name:     "every field",
			failover: mapperFullFailover(),
			want: map[string]interface{}{
				DNSZoneRecordSchemaFailoverProtocol:       "HTTP",
				DNSZoneRecordSchemaFailoverPort:           443,
				DNSZoneRecordSchemaFailoverFrequency:      30,
				DNSZoneRecordSchemaFailoverTimeout:        5,
				DNSZoneRecordSchemaFailoverMethod:         "GET",
				DNSZoneRecordSchemaFailoverURL:            "/health",
				DNSZoneRecordSchemaFailoverTLS:            true,
				DNSZoneRecordSchemaFailoverRegexp:         "^ok$",
				DNSZoneRecordSchemaFailoverHTTPStatusCode: 200,
				DNSZoneRecordSchemaFailoverHost:           "probe.example.com",
				DNSZoneRecordSchemaFailoverVerify:         true,
			},
		},
		{
			name:     "tls without verify emits both keys",
			failover: &dnssdk.FailoverMeta{Protocol: "HTTP", Tls: true},
			want: map[string]interface{}{
				DNSZoneRecordSchemaFailoverProtocol: "HTTP",
				DNSZoneRecordSchemaFailoverTLS:      true,
				DNSZoneRecordSchemaFailoverVerify:   false,
			},
		},
		{
			name:     "verify without tls is dropped",
			failover: &dnssdk.FailoverMeta{Protocol: "TCP", Verify: true},
			want:     map[string]interface{}{DNSZoneRecordSchemaFailoverProtocol: "TCP"},
		},
		{
			name: "zero valued fields are dropped one by one",
			failover: &dnssdk.FailoverMeta{
				Protocol:  "TCP",
				Port:      0,
				Frequency: 30,
				Timeout:   0,
				Method:    "",
				Url:       "",
				Regexp:    "",
				Host:      "",
			},
			want: map[string]interface{}{
				DNSZoneRecordSchemaFailoverProtocol:  "TCP",
				DNSZoneRecordSchemaFailoverFrequency: 30,
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := failoverMetaToList(&dnssdk.Meta{Failover: tt.failover})
			if len(got) != 1 {
				t.Fatalf("expected exactly one element, got %d: %#v", len(got), got)
			}

			outer, ok := got[0].(map[string][]interface{})
			if !ok {
				t.Fatalf("element type: %T, want map[string][]interface{}", got[0])
			}

			block, ok := outer[DNSZoneRecordSchemaFailover]
			if tt.want == nil {
				if ok {
					t.Fatalf("expected no failover key, got %#v", block)
				}

				return
			}
			if !ok {
				t.Fatalf("expected a failover key, got %#v", outer)
			}
			if len(block) != 1 {
				t.Fatalf("expected one failover block, got %#v", block)
			}
			if !reflect.DeepEqual(block[0], tt.want) {
				t.Fatalf("failover block:\n got: %#v\nwant: %#v", block[0], tt.want)
			}
		})
	}
}

func TestRecordVerifyFailoverMeta(t *testing.T) {
	t.Parallel()

	fields := []struct {
		name    string
		apply   func(*dnssdk.FailoverMeta)
		wantErr string
	}{
		{
			name:    DNSZoneRecordSchemaFailoverURL,
			apply:   func(f *dnssdk.FailoverMeta) { f.Url = "/health" },
			wantErr: "failover URL can only be set along with HTTP protocol",
		},
		{
			name:    DNSZoneRecordSchemaFailoverHost,
			apply:   func(f *dnssdk.FailoverMeta) { f.Host = "probe.example.com" },
			wantErr: "failover host can only be set along with HTTP protocol",
		},
		{
			name:    DNSZoneRecordSchemaFailoverRegexp,
			apply:   func(f *dnssdk.FailoverMeta) { f.Regexp = "^ok$" },
			wantErr: "failover regexp can only be set along with HTTP protocol",
		},
		{
			name:    DNSZoneRecordSchemaFailoverMethod,
			apply:   func(f *dnssdk.FailoverMeta) { f.Method = "GET" },
			wantErr: "failover method can only be set along with HTTP protocol",
		},
	}

	protocols := []struct {
		name     string
		protocol string
	}{
		{name: "TCP", protocol: "TCP"},
		{name: "UDP", protocol: "UDP"},
		{name: "ICMP", protocol: "ICMP"},
		{name: "empty", protocol: ""},
		{name: "lowercase http", protocol: "http"},
	}

	for _, p := range protocols {
		p := p
		for _, f := range fields {
			f := f
			t.Run(p.name+"/"+f.name, func(t *testing.T) {
				t.Parallel()

				failover := &dnssdk.FailoverMeta{Protocol: p.protocol, Port: 80, Frequency: 30, Timeout: 5}
				f.apply(failover)

				err := verifyFailoverMeta(dnssdk.Meta{Failover: failover})
				if err == nil {
					t.Fatalf("expected %q, got nil", f.wantErr)
				}
				if err.Error() != f.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), f.wantErr)
				}
			})
		}
	}

	t.Run("nil failover", func(t *testing.T) {
		t.Parallel()

		if err := verifyFailoverMeta(dnssdk.Meta{}); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("HTTP accepts every field", func(t *testing.T) {
		t.Parallel()

		if err := verifyFailoverMeta(dnssdk.Meta{Failover: mapperFullFailover()}); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("TCP without HTTP only fields", func(t *testing.T) {
		t.Parallel()

		failover := &dnssdk.FailoverMeta{Protocol: "TCP", Port: 80, Frequency: 30, Timeout: 5, Tls: true, Verify: true, HTTPStatusCode: 200}
		if err := verifyFailoverMeta(dnssdk.Meta{Failover: failover}); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("URL is reported first", func(t *testing.T) {
		t.Parallel()

		failover := &dnssdk.FailoverMeta{Protocol: "TCP", Url: "/health", Host: "probe.example.com", Regexp: "^ok$", Method: "GET"}
		err := verifyFailoverMeta(dnssdk.Meta{Failover: failover})
		if err == nil || err.Error() != "failover URL can only be set along with HTTP protocol" {
			t.Fatalf("error = %v, want the URL error", err)
		}
	})
}

func TestRecordFailoverMetaRoundTripPanicsWithoutState(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic, failoverMetaToList emits map[string][]interface{} but listToFailoverMeta asserts map[string]interface{}")
		}
	}()

	listToFailoverMeta(failoverMetaToList(&dnssdk.Meta{Failover: mapperFullFailover()}))
}

func TestRecordFailoverMetaRoundTripThroughState(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		failover *dnssdk.FailoverMeta
		want     *dnssdk.FailoverMeta
	}{
		{
			name:     "HTTP failover is stable",
			failover: mapperFullFailover(),
			want:     mapperFullFailover(),
		},
		{
			name:     "verify is lost when tls is off",
			failover: &dnssdk.FailoverMeta{Protocol: "TCP", Port: 80, Frequency: 30, Timeout: 5, Verify: true},
			want:     &dnssdk.FailoverMeta{Protocol: "TCP", Port: 80, Frequency: 30, Timeout: 5},
		},
		{
			name:     "an empty failover block collapses into no failover at all",
			failover: &dnssdk.FailoverMeta{},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := schema.TestResourceDataRaw(t, resourceDNSZoneRecord().Schema, map[string]interface{}{})
			if err := d.Set(DNSZoneRecordSchemaRRSetMeta, failoverMetaToList(&dnssdk.Meta{Failover: tt.failover})); err != nil {
				t.Fatalf("d.Set(%s): %v", DNSZoneRecordSchemaRRSetMeta, err)
			}

			got := listToFailoverMeta(d.Get(DNSZoneRecordSchemaRRSetMeta).([]interface{}))
			if !reflect.DeepEqual(got.Failover, tt.want) {
				t.Fatalf("failover:\n got: %#v\nwant: %#v", got.Failover, tt.want)
			}
		})
	}
}
