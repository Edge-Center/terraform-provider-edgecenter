//go:build integration

package dns_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dnsmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dns/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
)

const (
	dsZonesDataSourceName = "edgecenter_dns_secondary_zones"

	dsZonesPlaceholderID = "-"
	dsZonesReadID        = "secondary_zones"

	dsZonesCountAttr = dns.DNSSecondaryZonesSchemaZones + ".#"

	dsZonesTSIGZoneName  = "tsig.example.com"
	dsZonesTSIGZoneID    = 101
	dsZonesTSIGMaster    = "10.0.0.1"
	dsZonesTSIGName      = "keyname.tsig.example.com"
	dsZonesTSIGKey       = "c2VjcmV0LXRzaWcta2V5"
	dsZonesPlainZoneName = "plain.example.com"
	dsZonesPlainZoneID   = 202

	dsZonesKeyOnlyZoneName = "keyonly.example.com"
	dsZonesKeyOnlyZoneID   = 303
	dsZonesKeyOnlyTSIGName = "keyonly-key.example.com"

	dsZonesUpdatedAtNanos = 1700000000123456789
)

func dsZonesUpdatedAt() string {
	return dnssdk.Timestamp(dsZonesUpdatedAtNanos).Time().Format(time.RFC3339)
}

func dsZonesAttr(index int, field string) string {
	return fmt.Sprintf("%s.%d.%s", dns.DNSSecondaryZonesSchemaZones, index, field)
}

func dsZonesFields() []string {
	return []string{
		dns.DNSSecondaryZoneSchemaName,
		dns.DNSSecondaryZoneSchemaMaster,
		dns.DNSSecondaryZoneSchemaTSIGName,
		dns.DNSSecondaryZoneSchemaZoneID,
		dns.DNSSecondaryZoneSchemaUpdatedAt,
	}
}

func dsZonesStateFields(state *terraform.InstanceState, index int) []string {
	prefix := fmt.Sprintf("%s.%d.", dns.DNSSecondaryZonesSchemaZones, index)

	fields := make([]string, 0, len(state.Attributes))
	for key := range state.Attributes {
		if strings.HasPrefix(key, prefix) {
			fields = append(fields, strings.TrimPrefix(key, prefix))
		}
	}

	return fields
}

func dsZonesStaleState() map[string]interface{} {
	return map[string]interface{}{
		dns.DNSSecondaryZonesSchemaZones: []interface{}{
			map[string]interface{}{
				dns.DNSSecondaryZoneSchemaName:      "stale.example.com",
				dns.DNSSecondaryZoneSchemaZoneID:    999,
				dns.DNSSecondaryZoneSchemaMaster:    "192.0.2.99",
				dns.DNSSecondaryZoneSchemaTSIGName:  "stale-key.stale.example.com",
				dns.DNSSecondaryZoneSchemaUpdatedAt: "1999-01-01T00:00:00Z",
			},
		},
	}
}

func dsZonesSample() []dnssdk.SecondaryZone {
	return []dnssdk.SecondaryZone{
		{
			ID:   dsZonesTSIGZoneID,
			Name: dsZonesTSIGZoneName,
			TSIG: &dnssdk.TsigOptions{
				Key:    dsZonesTSIGKey,
				Master: dsZonesTSIGMaster,
				Name:   dsZonesTSIGName,
			},
			UpdatedAt: dsZonesUpdatedAtNanos,
		},
		{
			ID:   dsZonesPlainZoneID,
			Name: dsZonesPlainZoneName,
		},
	}
}

func dsZonesReadCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("SecondaryZones", mock.Anything).Return(dsZonesSample(), nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read replaces stale zones with the API list",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, dsZonesReadID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "2",
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaName):      dsZonesTSIGZoneName,
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaZoneID):    fmt.Sprintf("%d", dsZonesTSIGZoneID),
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaMaster):    dsZonesTSIGMaster,
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaTSIGName):  dsZonesTSIGName,
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaUpdatedAt): dsZonesUpdatedAt(),
				dsZonesAttr(1, dns.DNSSecondaryZoneSchemaName):      dsZonesPlainZoneName,
				dsZonesAttr(1, dns.DNSSecondaryZoneSchemaZoneID):    fmt.Sprintf("%d", dsZonesPlainZoneID),
				dsZonesAttr(1, dns.DNSSecondaryZoneSchemaMaster):    "",
				dsZonesAttr(1, dns.DNSSecondaryZoneSchemaTSIGName):  "",
				dsZonesAttr(1, dns.DNSSecondaryZoneSchemaUpdatedAt): "",
			})

			require.NotContains(t, state.Attributes[dsZonesAttr(0, dns.DNSSecondaryZoneSchemaUpdatedAt)], ".",
				"RFC3339 formatting drops the sub-second part of the timestamp")

			require.ElementsMatch(t, dsZonesFields(), dsZonesStateFields(state, 0),
				"a zone with TSIG must expose exactly the five documented fields")
			require.ElementsMatch(t, dsZonesFields(), dsZonesStateFields(state, 1),
				"a zone without TSIG must expose exactly the five documented fields")

			for key, value := range state.Attributes {
				require.NotContains(t, key, dns.DNSSecondaryZoneSchemaTSIGKey, "tsig_key must not reach the state")
				require.NotEqual(t, dsZonesTSIGKey, value, "the tsig key value must not reach the state")
				require.Truef(t, key == "id" || strings.HasPrefix(key, dns.DNSSecondaryZonesSchemaZones+"."),
					"zones is the only attribute the data source exposes, got %q", key)
			}
		},
	}
}

func dsZonesTSIGWithoutMasterCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("SecondaryZones", mock.Anything).Return([]dnssdk.SecondaryZone{
		{
			ID:   dsZonesKeyOnlyZoneID,
			Name: dsZonesKeyOnlyZoneName,
			TSIG: &dnssdk.TsigOptions{Key: dsZonesTSIGKey, Name: dsZonesKeyOnlyTSIGName},
		},
	}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "tsig without master still exposes tsig_name",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, dsZonesReadID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "1",
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaName):      dsZonesKeyOnlyZoneName,
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaZoneID):    fmt.Sprintf("%d", dsZonesKeyOnlyZoneID),
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaMaster):    "",
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaTSIGName):  dsZonesKeyOnlyTSIGName,
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaUpdatedAt): "",
			})

			require.ElementsMatch(t, dsZonesFields(), dsZonesStateFields(state, 0),
				"a zone with a master-less TSIG must expose exactly the five documented fields")

			for key, value := range state.Attributes {
				require.NotContains(t, key, dns.DNSSecondaryZoneSchemaTSIGKey, "tsig_key must not reach the state")
				require.NotEqual(t, dsZonesTSIGKey, value, "the tsig key value must not reach the state")
			}
		},
	}
}

func dsZonesEmptyListCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("SecondaryZones", mock.Anything).Return([]dnssdk.SecondaryZone{}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "empty zone list clears zones and still sets the id",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, dsZonesReadID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "0",
			})
			require.Empty(t, dsZonesStateFields(state, 0), "the stale element must be gone from the state")
		},
	}
}

func dsZonesNilListCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("SecondaryZones", mock.Anything).Return(nil, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "nil zone list behaves like an empty one",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, dsZonesReadID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "0",
			})
			require.Empty(t, dsZonesStateFields(state, 0), "the stale element must be gone from the state")
		},
	}
}

func dsZonesAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("SecondaryZones", mock.Anything).
		Return(nil, dnssdk.APIError{StatusCode: 503, Message: "server unavailable"})

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get secondary zones")
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			support.RequireStateID(t, state, dsZonesPlaceholderID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "1",
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaName): "stale.example.com",
			})
		},
	}
}

func dsZonesNilClientCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewUnconfiguredDNS()

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "nil dns client is rejected before the API call",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    dsZonesPlaceholderID,
		CurrentState: dsZonesStaleState(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "dns api client is null")
			support.RequireStateID(t, state, dsZonesPlaceholderID)
			support.RequireStateAttrs(t, state, map[string]string{
				dsZonesCountAttr: "1",
				dsZonesAttr(0, dns.DNSSecondaryZoneSchemaName): "stale.example.com",
			})
		},
	}
}

func TestIntegrationDataSourceSecondaryZones_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := dnsDataSource(t, dsZonesDataSourceName)

	cases := []support.ResourceCase[*dnsmock.MockedDNS]{
		dsZonesReadCase(),
		dsZonesTSIGWithoutMasterCase(),
		dsZonesEmptyListCase(),
		dsZonesNilListCase(),
		dsZonesAPIFailureCase(),
		dsZonesNilClientCase(),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*dnsmock.MockedDNS])
}
