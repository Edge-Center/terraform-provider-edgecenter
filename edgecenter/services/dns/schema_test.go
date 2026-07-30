package dns

import (
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type schemaValidateCase struct {
	name            string
	value           interface{}
	wantErr         bool
	wantErrContains string
}

func schemaRunValidateCases(t *testing.T, validate schema.SchemaValidateDiagFunc, cases []schemaValidateCase) {
	t.Helper()

	if validate == nil {
		t.Fatal("ValidateDiagFunc is nil")
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantErr == (tt.wantErrContains == "") {
				t.Fatalf("case setup: wantErr = %v, wantErrContains = %q", tt.wantErr, tt.wantErrContains)
			}

			diags := validate(tt.value, cty.Path{})

			if !tt.wantErr {
				if diags.HasError() {
					t.Fatalf("validate(%#v) returned %v, want no error", tt.value, diags)
				}

				return
			}

			if !diags.HasError() {
				t.Fatalf("validate(%#v) returned no error, want one", tt.value)
			}
			if len(diags) != 1 {
				t.Fatalf("validate(%#v) returned %d diagnostics, want 1: %v", tt.value, len(diags), diags)
			}
			if !strings.Contains(diags[0].Summary, tt.wantErrContains) {
				t.Fatalf("validate(%#v) summary = %q, want it to contain %q", tt.value, diags[0].Summary, tt.wantErrContains)
			}
		})
	}
}

func schemaNestedField(t *testing.T, s *schema.Schema, key string) *schema.Schema {
	t.Helper()

	res, ok := s.Elem.(*schema.Resource)
	if !ok {
		t.Fatalf("elem is %T, want *schema.Resource", s.Elem)
	}
	field, ok := res.Schema[key]
	if !ok {
		t.Fatalf("nested field %q is missing", key)
	}

	return field
}

func schemaElemField(t *testing.T, s *schema.Schema) *schema.Schema {
	t.Helper()

	elem, ok := s.Elem.(*schema.Schema)
	if !ok {
		t.Fatalf("elem is %T, want *schema.Schema", s.Elem)
	}

	return elem
}

func schemaRecordMetaField(t *testing.T, key string) *schema.Schema {
	t.Helper()

	record := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaResourceRecord]

	return schemaNestedField(t, schemaNestedField(t, record, DNSZoneRecordSchemaMeta), key)
}

func schemaKeys(m map[string]*schema.Resource) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func TestSchemaServiceRegistersAllDNSNames(t *testing.T) {
	t.Parallel()

	svc := Service{}

	if svc.Name() != "dns" {
		t.Fatalf("Name() = %q, want dns", svc.Name())
	}

	wantResources := []string{"edgecenter_dns_secondary_zone", "edgecenter_dns_zone", "edgecenter_dns_zone_record"}
	sort.Strings(wantResources)
	if got := schemaKeys(svc.Resources()); !schemaEqualStrings(got, wantResources) {
		t.Fatalf("Resources() = %v, want %v", got, wantResources)
	}

	wantDataSources := []string{"edgecenter_dns_secondary_zones"}
	if got := schemaKeys(svc.DataSources()); !schemaEqualStrings(got, wantDataSources) {
		t.Fatalf("DataSources() = %v, want %v", got, wantDataSources)
	}
}

func schemaEqualStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestSchemaResourcesPassInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		name, res := name, res
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := res.InternalValidate(nil, true); err != nil {
				t.Fatalf("InternalValidate() failed: %v", err)
			}
		})
	}
}

func TestSchemaDataSourcesPassInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).DataSources() {
		name, res := name, res
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := res.InternalValidate(nil, false); err != nil {
				t.Fatalf("InternalValidate() failed: %v", err)
			}
		})
	}
}

func TestSchemaCRUDContexts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		res          *schema.Resource
		wantCreate   bool
		wantRead     bool
		wantUpdate   bool
		wantDelete   bool
		wantImporter bool
	}{
		{DNSZoneResource, resourceDNSZone(), true, true, false, true, true},
		{DNSZoneRecordResource, resourceDNSZoneRecord(), true, true, true, true, true},
		{DNSSecondaryZoneResource, resourceDNSSecondaryZone(), true, true, true, true, true},
		{DNSSecondaryZonesDataSource, dataSourceDNSSecondaryZones(), false, true, false, false, false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if (tt.res.CreateContext != nil) != tt.wantCreate {
				t.Fatalf("CreateContext set = %v, want %v", tt.res.CreateContext != nil, tt.wantCreate)
			}
			if (tt.res.ReadContext != nil) != tt.wantRead {
				t.Fatalf("ReadContext set = %v, want %v", tt.res.ReadContext != nil, tt.wantRead)
			}
			if (tt.res.UpdateContext != nil) != tt.wantUpdate {
				t.Fatalf("UpdateContext set = %v, want %v", tt.res.UpdateContext != nil, tt.wantUpdate)
			}
			if (tt.res.DeleteContext != nil) != tt.wantDelete {
				t.Fatalf("DeleteContext set = %v, want %v", tt.res.DeleteContext != nil, tt.wantDelete)
			}
			if (tt.res.Importer != nil) != tt.wantImporter {
				t.Fatalf("Importer set = %v, want %v", tt.res.Importer != nil, tt.wantImporter)
			}
			if tt.res.Description == "" {
				t.Fatal("Description is empty")
			}
		})
	}
}

func TestSchemaZoneHasNoUpdateAndIsFullyForceNew(t *testing.T) {
	t.Parallel()

	res := resourceDNSZone()
	if res.UpdateContext != nil {
		t.Fatal("edgecenter_dns_zone must not define UpdateContext")
	}

	for name, field := range res.Schema {
		if !field.ForceNew && !field.Computed {
			t.Fatalf("field %q must be ForceNew, the resource has no UpdateContext", name)
		}
	}
}

func TestSchemaForceNewFields(t *testing.T) {
	t.Parallel()

	record := resourceDNSZoneRecord().Schema
	secondary := resourceDNSSecondaryZone().Schema

	cases := []struct {
		name     string
		field    *schema.Schema
		forceNew bool
	}{
		{"zone name", resourceDNSZone().Schema[DNSZoneSchemaName], true},
		{"record zone", record[DNSZoneRecordSchemaZone], true},
		{"record domain", record[DNSZoneRecordSchemaDomain], true},
		{"record type", record[DNSZoneRecordSchemaType], true},
		{"record ttl", record[DNSZoneRecordSchemaTTL], false},
		{"record resource_record", record[DNSZoneRecordSchemaResourceRecord], false},
		{"secondary zone name", secondary[DNSSecondaryZoneSchemaName], true},
		{"secondary zone master", secondary[DNSSecondaryZoneSchemaMaster], false},
		{"secondary zone tsig_key", secondary[DNSSecondaryZoneSchemaTSIGKey], false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.field == nil {
				t.Fatal("field is missing from the schema")
			}
			if tt.field.ForceNew != tt.forceNew {
				t.Fatalf("ForceNew = %v, want %v", tt.field.ForceNew, tt.forceNew)
			}
		})
	}
}

func TestSchemaZoneNameValidation(t *testing.T) {
	t.Parallel()

	schemaRunValidateCases(t, resourceDNSZone().Schema[DNSZoneSchemaName].ValidateDiagFunc, []schemaValidateCase{
		{"blank", "", true, "dns name can't be empty"},
		{"whitespace only", "   ", true, "dns name can't be empty"},
		{"256 ascii chars", strings.Repeat("a", 256), true, "should be less than 256 symbols"},
		{"255 ascii chars", strings.Repeat("a", 255), false, ""},
		{"200 multibyte runes exceed the byte limit", strings.Repeat(string(rune(0xE9)), 200), true, "should be less than 256 symbols"},
		{"normal name", "example.com", false, ""},
		{"padded name", "  example.com  ", false, ""},
		{"name without a dot", "localhost", false, ""},
	})
}

func TestSchemaRecordZoneValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaZone]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"blank", "", true, "dns record zone can't be empty"},
		{"whitespace only", " \t ", true, "dns record zone can't be empty"},
		{"256 ascii chars", strings.Repeat("a", 256), true, "should be less than 256 symbols"},
		{"255 ascii chars", strings.Repeat("a", 255), false, ""},
		{"normal zone", "example.com", false, ""},
		{"zone without a dot", "localhost", false, ""},
	})
}

func TestSchemaRecordDomainValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaDomain]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"blank", "", true, "dns record domain can't be empty"},
		{"whitespace only", " \t ", true, "dns record domain can't be empty"},
		{"256 ascii chars", strings.Repeat("a", 256), true, "should be less than 256 symbols"},
		{"255 ascii chars", strings.Repeat("a", 255), false, ""},
		{"normal domain", "www.example.com", false, ""},
		{"wildcard domain", "*.example.com", false, ""},
	})
}

func TestSchemaSecondaryZoneNameValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSSecondaryZone().Schema[DNSSecondaryZoneSchemaName]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"blank", "", true, "secondary zone name can't be empty"},
		{"whitespace only", " ", true, "secondary zone name can't be empty"},
		{"no dot", "localhost", true, "secondary zone name must be a valid domain name"},
		{"too long", strings.Repeat("a", 252) + ".com", true, "should be less than 256 symbols"},
		{"valid domain", "example.com", false, ""},
		{"trailing dot only", ".", false, ""},
	})
}

func TestSchemaSecondaryZoneMasterValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSSecondaryZone().Schema[DNSSecondaryZoneSchemaMaster]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"blank", "", true, "master server address can't be empty"},
		{"whitespace only", "\t", true, "master server address can't be empty"},
		{"no dot", "ns1", true, "master server must be a valid IP address or domain name"},
		{"ipv6 has no dot", "2001:db8::1", true, "master server must be a valid IP address or domain name"},
		{"ipv4", "10.0.0.1", false, ""},
		{"hostname", "ns1.example.com", false, ""},
		{"out of range octets are still accepted", "999.999.999.999", false, ""},
	})
}

func TestSchemaRecordTypeValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaType]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"a", "A", false, ""},
		{"aaaa", "AAAA", false, ""},
		{"mx", "MX", false, ""},
		{"cname", "CNAME", false, ""},
		{"txt", "TXT", false, ""},
		{"caa", "CAA", false, ""},
		{"ns", "NS", false, ""},
		{"srv", "SRV", false, ""},
		{"dname", "DNAME", false, ""},
		{"lowercase", "cname", false, ""},
		{"mixed case", "aAaA", false, ""},
		{"padded", "  mx  ", false, ""},
		{"empty", "", true, "dns record type should be one of [A AAAA MX CNAME TXT CAA NS SRV DNAME]"},
		{"ptr is not supported", "PTR", true, "dns record type should be one of [A AAAA MX CNAME TXT CAA NS SRV DNAME]"},
		{"soa is not supported", "SOA", true, "dns record type should be one of [A AAAA MX CNAME TXT CAA NS SRV DNAME]"},
	})
}

func TestSchemaRecordTTLValidation(t *testing.T) {
	t.Parallel()

	field := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaTTL]

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"negative", -1, true, "dns record ttl can't be less than 0"},
		{"zero", 0, false, ""},
		{"positive", 3600, false, ""},
		{"no upper bound", 1 << 40, false, ""},
	})
}

func TestSchemaRecordFilterTypeValidation(t *testing.T) {
	t.Parallel()

	filter := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaFilter]
	field := schemaNestedField(t, filter, DNSZoneRecordSchemaFilterType)

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"geodns", "geodns", false, ""},
		{"geodistance", "geodistance", false, ""},
		{"default", "default", false, ""},
		{"first n", "first_n", false, ""},
		{"is healthy", "is_healthy", false, ""},
		{"empty", "", true, "dns record filter type should be one of [geodns geodistance default first_n is_healthy]"},
		{"unknown", "roundrobin", true, "dns record filter type should be one of [geodns geodistance default first_n is_healthy]"},
		{"uppercase is rejected", "GEODNS", true, "dns record filter type should be one of"},
		{"padded is rejected", " geodns ", true, "dns record filter type should be one of"},
	})
}

func TestSchemaFailoverProtocolValidation(t *testing.T) {
	t.Parallel()

	rrsetMeta := resourceDNSZoneRecord().Schema[DNSZoneRecordSchemaRRSetMeta]
	failover := schemaNestedField(t, rrsetMeta, DNSZoneRecordSchemaFailover)
	field := schemaNestedField(t, failover, DNSZoneRecordSchemaFailoverProtocol)

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"tcp", "TCP", false, ""},
		{"udp", "UDP", false, ""},
		{"http", "HTTP", false, ""},
		{"icmp", "ICMP", false, ""},
		{"lowercase", "http", false, ""},
		{"mixed case", "IcMp", false, ""},
		{"padded", " tcp ", false, ""},
		{"empty", "", true, "dns failover protocol type should be one of [TCP UDP HTTP ICMP]"},
		{"https is not a protocol here", "HTTPS", true, "dns failover protocol type should be one of [TCP UDP HTTP ICMP]"},
	})
}

func TestSchemaRecordMetaAsnValidation(t *testing.T) {
	t.Parallel()

	field := schemaElemField(t, schemaRecordMetaField(t, DNSZoneRecordSchemaMetaAsn))

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"negative", -1, true, "asn cannot be less then 0"},
		{"zero", 0, false, ""},
		{"positive", 12345, false, ""},
	})
}

func TestSchemaRecordMetaIPValidation(t *testing.T) {
	t.Parallel()

	field := schemaElemField(t, schemaRecordMetaField(t, DNSZoneRecordSchemaMetaIP))

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"empty", "", true, "dns record meta ip has wrong format: "},
		{"not an ip", "not-an-ip", true, "dns record meta ip has wrong format: not-an-ip"},
		{"cidr", "127.0.0.1/32", true, "dns record meta ip has wrong format: 127.0.0.1/32"},
		{"ipv4", "127.0.0.1", false, ""},
		{"ipv6", "2001:db8::1", false, ""},
	})
}

func TestSchemaRecordMetaWeightValidation(t *testing.T) {
	t.Parallel()

	field := schemaRecordMetaField(t, DNSZoneRecordSchemaMetaWeight)

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"below zero", -1, true, "weight must be between 0 and 100"},
		{"above one hundred", 101, true, "weight must be between 0 and 100"},
		{"zero", 0, false, ""},
		{"one hundred", 100, false, ""},
		{"middle", 50, false, ""},
	})
}

func TestSchemaRecordMetaRegionsValidation(t *testing.T) {
	t.Parallel()

	const validRegion = "ru-spb"

	if _, ok := validDNSRecordRegions[validRegion]; !ok {
		t.Fatalf("region %q is missing from validDNSRecordRegions", validRegion)
	}

	field := schemaElemField(t, schemaRecordMetaField(t, DNSZoneRecordSchemaMetaRegions))

	schemaRunValidateCases(t, field.ValidateDiagFunc, []schemaValidateCase{
		{"known region", validRegion, false, ""},
		{"unknown region", "ru-xxx", true, `unsupported dns record region "ru-xxx"`},
		{"uppercase region", strings.ToUpper(validRegion), true, `unsupported dns record region "RU-SPB"`},
		{"empty", "", true, `unsupported dns record region ""`},
	})
}

func TestSchemaValidDNSRecordRegions(t *testing.T) {
	t.Parallel()

	if len(validDNSRecordRegions) == 0 {
		t.Fatal("validDNSRecordRegions must not be empty")
	}

	for region := range validDNSRecordRegions {
		if region == "" {
			t.Fatal("validDNSRecordRegions contains an empty key")
		}
		if region != strings.ToLower(region) {
			t.Fatalf("region %q must be lowercase", region)
		}
	}
}

func TestSchemaSecondaryZoneTSIGKeyIsSensitive(t *testing.T) {
	t.Parallel()

	res := resourceDNSSecondaryZone()

	key := res.Schema[DNSSecondaryZoneSchemaTSIGKey]
	if key == nil {
		t.Fatal("tsig_key is missing from the secondary zone schema")
	}
	if !key.Sensitive {
		t.Fatal("tsig_key must be Sensitive")
	}

	if name := res.Schema[DNSSecondaryZoneSchemaTSIGName]; name == nil || name.Sensitive {
		t.Fatalf("tsig_name = %+v, want a non sensitive field", name)
	}
}

func TestSchemaSecondaryZonesDataSourceHidesTSIGKey(t *testing.T) {
	t.Parallel()

	zones := dataSourceDNSSecondaryZones().Schema[DNSSecondaryZonesSchemaZones]
	if zones == nil {
		t.Fatal("zones is missing from the data source schema")
	}
	if !zones.Computed {
		t.Fatal("zones must be Computed")
	}

	elem, ok := zones.Elem.(*schema.Resource)
	if !ok {
		t.Fatalf("zones elem is %T, want *schema.Resource", zones.Elem)
	}

	if _, ok := elem.Schema[DNSSecondaryZoneSchemaTSIGKey]; ok {
		t.Fatal("the data source must not expose tsig_key")
	}

	for _, name := range []string{
		DNSSecondaryZoneSchemaName,
		DNSSecondaryZoneSchemaMaster,
		DNSSecondaryZoneSchemaTSIGName,
		DNSSecondaryZoneSchemaZoneID,
		DNSSecondaryZoneSchemaUpdatedAt,
	} {
		field, ok := elem.Schema[name]
		if !ok {
			t.Fatalf("the data source must expose %q", name)
		}
		if !field.Computed {
			t.Fatalf("data source field %q must be Computed", name)
		}
	}
}
