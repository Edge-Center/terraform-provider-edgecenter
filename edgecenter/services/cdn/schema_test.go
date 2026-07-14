package cdn

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cdn "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/origingroups"
	"github.com/Edge-Center/edgecentercdn-go/shielding"
)

func TestStructToMap(t *testing.T) {
	t.Parallel()

	got := structToMap(&cdn.BrowserCacheSettings{Enabled: true, Value: "3600s"})
	want := map[string]interface{}{
		"enabled": true,
		"value":   "3600s",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("structToMap() = %v, want %v", got, want)
	}
}

func TestStructToMapNil(t *testing.T) {
	t.Parallel()

	if got := structToMap(nil); len(got) != 0 {
		t.Fatalf("structToMap(nil) = %v, want empty map", got)
	}
}

func TestGetLocationByDC(t *testing.T) {
	t.Parallel()

	locations := []shielding.ShieldingLocations{
		{ID: 18, Datacenter: "dc-2"},
		{ID: 17, Datacenter: "dc-1"},
	}

	id, err := getLocationByDC(locations, "dc-1")
	if err != nil {
		t.Fatalf("getLocationByDC() unexpected error: %v", err)
	}
	if id != 17 {
		t.Fatalf("getLocationByDC() = %d, want 17", id)
	}

	if _, err := getLocationByDC(locations, "dc-missing"); err == nil {
		t.Fatal("getLocationByDC() expected error for unknown datacenter")
	}
}

func TestOriginSetIDFuncIsDeterministic(t *testing.T) {
	t.Parallel()

	fields := map[string]interface{}{
		"id":      1,
		"source":  "example.com",
		"enabled": true,
		"backup":  false,
	}

	first := originSetIDFunc(fields)
	if second := originSetIDFunc(fields); first != second {
		t.Fatalf("originSetIDFunc() is not deterministic: %d != %d", first, second)
	}

	other := originSetIDFunc(map[string]interface{}{
		"id":      1,
		"source":  "other.example.com",
		"enabled": true,
		"backup":  false,
	})
	if first == other {
		t.Fatal("originSetIDFunc() must differ for a different source")
	}
}

func TestOriginsRoundTrip(t *testing.T) {
	t.Parallel()

	origins := []origingroups.Origin{
		{ID: 1, Source: "a.example.com", Enabled: true, Backup: false},
		{ID: 2, Source: "b.example.com", Enabled: false, Backup: true},
	}

	got := setToOriginRequests(originsToSet(origins))
	if len(got) != 2 {
		t.Fatalf("setToOriginRequests() returned %d origins, want 2", len(got))
	}

	bySource := map[string]origingroups.OriginRequest{}
	for _, o := range got {
		bySource[o.Source] = o
	}

	if o := bySource["a.example.com"]; !o.Enabled || o.Backup {
		t.Fatalf("origin a.example.com = %+v, want enabled and not backup", o)
	}
	if o := bySource["b.example.com"]; o.Enabled || !o.Backup {
		t.Fatalf("origin b.example.com = %+v, want disabled and backup", o)
	}
}

func TestAuthRoundTrip(t *testing.T) {
	t.Parallel()

	auth := &origingroups.Authorization{
		AuthType:        "aws_signature_v4",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		AddressingStyle: "path",
		AwsRegion:       "us-east-1",
		SecretKey:       "secret",
		BucketName:      "bucket",
	}

	got := setToAuthRequest(authToSet(auth))
	if !reflect.DeepEqual(got, auth) {
		t.Fatalf("setToAuthRequest(authToSet(auth)) = %+v, want %+v", got, auth)
	}
}

func TestAuthToSetNil(t *testing.T) {
	t.Parallel()

	if got := authToSet(nil); got != nil {
		t.Fatalf("authToSet(nil) = %v, want nil", got)
	}
}

func TestSetToAuthRequestEmpty(t *testing.T) {
	t.Parallel()

	if got := setToAuthRequest(&schema.Set{F: schema.HashString}); got != nil {
		t.Fatalf("setToAuthRequest(empty) = %v, want nil", got)
	}
}

func TestListToResourceOptionsEmpty(t *testing.T) {
	t.Parallel()

	if got := listToResourceOptions(nil); got != nil {
		t.Fatalf("listToResourceOptions(nil) = %v, want nil", got)
	}
}

func TestOptionsConvertersTolerateNil(t *testing.T) {
	t.Parallel()

	if got := resourceOptionsToList(nil); got != nil {
		t.Fatalf("resourceOptionsToList(nil) = %v, want nil", got)
	}
	if got := locationOptionsToList(nil); got != nil {
		t.Fatalf("locationOptionsToList(nil) = %v, want nil", got)
	}
	if got := listToResourceOptions([]interface{}{nil}); got != nil {
		t.Fatalf("listToResourceOptions([nil]) = %v, want nil", got)
	}
	if got := listToLocationOptions([]interface{}{nil}); got != nil {
		t.Fatalf("listToLocationOptions([nil]) = %v, want nil", got)
	}
}

// resourceOptionsToList feeds d.Set and listToResourceOptions reads d.Get, so the
// two are only inverse through the schema layer, never as plain Go values.
func TestResourceOptionsRoundTripThroughSchema(t *testing.T) {
	t.Parallel()

	options := &cdn.ResourceOptions{
		LocationOptions: cdn.LocationOptions{
			BrowserCacheSettings: &cdn.BrowserCacheSettings{Enabled: true, Value: "3600s"},
			EdgeCacheSettings:    &cdn.EdgeCacheSettings{Enabled: true, Value: "43200s"},
			AllowedHTTPMethods:   &cdn.AllowedHTTPMethods{Enabled: true, Value: []string{"GET", "HEAD"}},
		},
	}

	data := schema.TestResourceDataRaw(t, resourceCDNResource().Schema, map[string]interface{}{})
	if err := data.Set("options", resourceOptionsToList(options)); err != nil {
		t.Fatalf("d.Set(options) failed: %v", err)
	}

	got := listToResourceOptions(data.Get("options").([]interface{}))

	if got == nil || got.BrowserCacheSettings == nil {
		t.Fatal("listToResourceOptions() lost browser_cache_settings")
	}
	if !got.BrowserCacheSettings.Enabled || got.BrowserCacheSettings.Value != "3600s" {
		t.Fatalf("browser_cache_settings = %+v, want enabled 3600s", got.BrowserCacheSettings)
	}
	if got.EdgeCacheSettings == nil || got.EdgeCacheSettings.Value != "43200s" {
		t.Fatalf("edge_cache_settings = %+v, want 43200s", got.EdgeCacheSettings)
	}
	if got.AllowedHTTPMethods == nil || len(got.AllowedHTTPMethods.Value) != 2 {
		t.Fatalf("allowed_http_methods = %+v, want 2 values", got.AllowedHTTPMethods)
	}
}

func TestListToLocationOptionsEmpty(t *testing.T) {
	t.Parallel()

	if got := listToLocationOptions(nil); got != nil {
		t.Fatalf("listToLocationOptions(nil) = %v, want nil", got)
	}
}

func TestLocationOptionsRoundTripThroughSchema(t *testing.T) {
	t.Parallel()

	options := &cdn.LocationOptions{
		BrowserCacheSettings: &cdn.BrowserCacheSettings{Enabled: true, Value: "600s"},
		IgnoreQueryString:    &cdn.IgnoreQueryString{Enabled: true, Value: true},
	}

	data := schema.TestResourceDataRaw(t, resourceCDNRule().Schema, map[string]interface{}{})
	if err := data.Set("options", locationOptionsToList(options)); err != nil {
		t.Fatalf("d.Set(options) failed: %v", err)
	}

	got := listToLocationOptions(data.Get("options").([]interface{}))

	if got == nil || got.BrowserCacheSettings == nil {
		t.Fatal("listToLocationOptions() lost browser_cache_settings")
	}
	if got.BrowserCacheSettings.Value != "600s" {
		t.Fatalf("browser_cache_settings.value = %q, want 600s", got.BrowserCacheSettings.Value)
	}
	if got.IgnoreQueryString == nil || !got.IgnoreQueryString.Value {
		t.Fatalf("ignore_query_string = %+v, want enabled true", got.IgnoreQueryString)
	}
}

func TestServiceRegistersAllCDNNames(t *testing.T) {
	t.Parallel()

	svc := Service{}

	if svc.Name() != "cdn" {
		t.Fatalf("Name() = %q, want cdn", svc.Name())
	}

	wantResources := []string{
		"edgecenter_cdn_resource",
		"edgecenter_cdn_origingroup",
		"edgecenter_cdn_lecert",
		"edgecenter_cdn_rule",
		"edgecenter_cdn_shielding",
		"edgecenter_cdn_sslcert",
	}

	resources := svc.Resources()
	if len(resources) != len(wantResources) {
		t.Fatalf("Resources() has %d entries, want %d", len(resources), len(wantResources))
	}
	for _, name := range wantResources {
		if resources[name] == nil {
			t.Fatalf("resource %q is not registered", name)
		}
	}

	wantDataSources := []string{
		"edgecenter_cdn_client_info",
		"edgecenter_cdn_shielding_location",
	}

	dataSources := svc.DataSources()
	if len(dataSources) != len(wantDataSources) {
		t.Fatalf("DataSources() has %d entries, want %d", len(dataSources), len(wantDataSources))
	}
	for _, name := range wantDataSources {
		if dataSources[name] == nil {
			t.Fatalf("data source %q is not registered", name)
		}
	}
}
