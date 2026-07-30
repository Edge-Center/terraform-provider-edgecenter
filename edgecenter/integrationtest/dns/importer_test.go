//go:build integration

package dns_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
)

const (
	importerRecordFormatErr = "format must be as zone:domain:type"

	importerPassthroughID = "  zone:with:colons.example.com  "
)

func TestIntegrationImporterZoneRecord(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, dns.DNSZoneRecordResource)
	require.NotNil(t, resource.Importer, "%s must be importable", dns.DNSZoneRecordResource)

	cases := []struct {
		name       string
		id         string
		wantErr    string
		wantID     string
		wantZone   string
		wantDomain string
		wantType   string
	}{
		{
			name:       "zone:domain:type is split and the id collapses to the zone",
			id:         "example.com:www:A",
			wantID:     "example.com",
			wantZone:   "example.com",
			wantDomain: "www",
			wantType:   "A",
		},
		{
			name:       "type is taken verbatim and is not validated",
			id:         "example.com:www:not-a-record-type",
			wantID:     "example.com",
			wantZone:   "example.com",
			wantDomain: "www",
			wantType:   "not-a-record-type",
		},
		{
			name:       "empty domain and type parts are accepted",
			id:         "example.com::",
			wantID:     "example.com",
			wantZone:   "example.com",
			wantDomain: "",
			wantType:   "",
		},
		{
			name:    "two parts are rejected",
			id:      "example.com:www",
			wantErr: importerRecordFormatErr,
		},
		{
			name:    "four parts are rejected",
			id:      "example.com:www:A:extra",
			wantErr: importerRecordFormatErr,
		},
		{
			name:    "empty id is rejected",
			id:      "",
			wantErr: importerRecordFormatErr,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := resource.Data(&terraform.InstanceState{
				ID: tt.id,
				Attributes: map[string]string{
					"zone":   "stale.example.com",
					"domain": "stale",
					"type":   "TXT",
				},
			})

			results, err := resource.Importer.StateContext(context.Background(), data, nil)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, results)

				return
			}

			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Equal(t, tt.wantID, results[0].Id())
			require.Equal(t, tt.wantZone, results[0].Get("zone").(string))
			require.Equal(t, tt.wantDomain, results[0].Get("domain").(string))
			require.Equal(t, tt.wantType, results[0].Get("type").(string))
		})
	}
}

func TestIntegrationImporterPassthroughZones(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		seed map[string]string
		want map[string]interface{}
	}{
		{
			name: dns.DNSZoneResource,
			seed: map[string]string{"name": "seeded.example.com"},
			want: map[string]interface{}{"name": "seeded.example.com"},
		},
		{
			name: dns.DNSSecondaryZoneResource,
			seed: map[string]string{
				"name":       "seeded.example.com",
				"master":     "10.0.0.1",
				"tsig_key":   "c2VjcmV0",
				"tsig_name":  "key.seeded.example.com",
				"zone_id":    "42",
				"updated_at": "2026-07-29T00:00:00Z",
			},
			want: map[string]interface{}{
				"name":       "seeded.example.com",
				"master":     "10.0.0.1",
				"tsig_key":   "c2VjcmV0",
				"tsig_name":  "key.seeded.example.com",
				"zone_id":    42,
				"updated_at": "2026-07-29T00:00:00Z",
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := dnsResource(t, tt.name)
			require.NotNil(t, resource.Importer, "%s must be importable", tt.name)
			require.NotNil(t, resource.Importer.StateContext, "%s must have an import state function", tt.name)

			data := resource.Data(&terraform.InstanceState{ID: importerPassthroughID, Attributes: tt.seed})

			results, err := resource.Importer.StateContext(context.Background(), data, nil)
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Same(t, data, results[0])
			require.Equal(t, importerPassthroughID, results[0].Id())

			for attr, want := range tt.want {
				require.Equal(t, want, results[0].Get(attr), "%s must survive the import untouched", attr)
			}

			require.Equal(t,
				reflect.ValueOf(schema.ImportStatePassthroughContext).Pointer(),
				reflect.ValueOf(resource.Importer.StateContext).Pointer(),
				"%s must import through the sdk passthrough", tt.name)
		})
	}
}
