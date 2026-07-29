//go:build integration

package dns_test

import (
	"context"
	"fmt"
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
	dnssvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
)

const (
	zoneName        = "example.com"
	zoneMixedCase   = "Example.COM"
	zonePadded      = "  padded.example.com  "
	zoneTrimmed     = "padded.example.com"
	zoneStale       = "stale.example.com"
	zoneRenamed     = "renamed.example.com"
	zoneReplacement = "other.example.com"
	zoneID          = uint64(42)
)

func zoneConfig(name string) map[string]interface{} {
	return map[string]interface{}{dnssvc.DNSZoneSchemaName: name}
}

// Create ids the resource by the requested name, then chains into Read, so the
// name that lands in state is the one Zone() returns, not the configured one.
func zoneCreateCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("CreateZone", mock.Anything, zoneMixedCase).Return(zoneID, nil)
	mc.Client.On("Zone", mock.Anything, zoneMixedCase).Return(dnssdk.Zone{Name: zoneName}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "successful create reads the zone back",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: zoneConfig(zoneMixedCase),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, zoneName)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zoneName})
		},
	}
}

func zoneCreateTrimsNameCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("CreateZone", mock.Anything, zoneTrimmed).Return(zoneID, nil)
	mc.Client.On("Zone", mock.Anything, zoneTrimmed).Return(dnssdk.Zone{Name: zoneTrimmed}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create trims surrounding spaces from the name",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: zoneConfig(zonePadded),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, zoneTrimmed)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zoneTrimmed})
		},
	}
}

func zoneCreateAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("CreateZone", mock.Anything, zoneName).
		Return(uint64(0), fmt.Errorf("api error: zone already exists"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: zoneConfig(zoneName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create zone")
			support.RequireErrorDiagContains(t, diags, "zone already exists")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

// The trimmed name becomes the id before the read runs, so a zone created on the
// API side stays in state even though the apply reports an error. The name
// attribute keeps the untrimmed configured value because Read never sets it.
func zoneCreateReadFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("CreateZone", mock.Anything, zoneTrimmed).Return(zoneID, nil)
	mc.Client.On("Zone", mock.Anything, zoneTrimmed).
		Return(dnssdk.Zone{}, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "read failure after create keeps the id in state",
		Op:        support.OpApply,
		Prepare:   func() *dnsmock.MockedDNS { return mc },
		NewConfig: zoneConfig(zonePadded),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get zone")
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			support.RequireStateID(t, state, zoneTrimmed)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zonePadded})
		},
	}
}

func zoneReadCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, zoneName).Return(dnssdk.Zone{Name: zoneRenamed}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "read overwrites id and name with API values",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    zoneName,
		CurrentState: zoneConfig(zoneStale),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, zoneRenamed)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zoneRenamed})
		},
	}
}

// A zone deleted out of band is an error, not a removal from state: Read never
// calls d.SetId("").
func zoneReadAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("Zone", mock.Anything, zoneName).
		Return(dnssdk.Zone{}, fmt.Errorf("api error: zone not found"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    zoneName,
		CurrentState: zoneConfig(zoneStale),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "get zone")
			support.RequireErrorDiagContains(t, diags, "zone not found")
			support.RequireStateID(t, state, zoneName)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zoneStale})
		},
	}
}

func zoneRenameRecreatesCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteZone", mock.Anything, zoneName).Return(nil)
	mc.Client.On("CreateZone", mock.Anything, zoneReplacement).Return(zoneID, nil)
	mc.Client.On("Zone", mock.Anything, zoneReplacement).Return(dnssdk.Zone{Name: zoneReplacement}, nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "rename destroys the old zone and creates the new one",
		Op:           support.OpApply,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    zoneName,
		CurrentState: zoneConfig(zoneName),
		NewConfig:    zoneConfig(zoneReplacement),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, zoneReplacement)
			support.RequireStateAttrs(t, state, map[string]string{dnssvc.DNSZoneSchemaName: zoneReplacement})
		},
	}
}

func zoneDeleteCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteZone", mock.Anything, zoneName).Return(nil)

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "delete clears the id",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    zoneName,
		CurrentState: zoneConfig(zoneStale),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func zoneDeleteAPIFailureCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("DeleteZone", mock.Anything, zoneName).
		Return(fmt.Errorf("api error: zone still has records"))

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *dnsmock.MockedDNS { return mc },
		CurrentID:    zoneName,
		CurrentState: zoneConfig(zoneName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "delete zone")
			support.RequireErrorDiagContains(t, diags, "still has records")
			support.RequireStateID(t, state, zoneName)
		},
	}
}

func zoneDeleteEmptyNameCase() support.ResourceCase[*dnsmock.MockedDNS] {
	mc := dnsmock.NewMockedDNS()

	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:    "delete without an id or a name never reaches the client",
		Op:      support.OpDelete,
		Prepare: func() *dnsmock.MockedDNS { return mc },
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "empty zone name")
			require.Nil(t, state)
			require.Empty(t, fake.Client.Calls, "client must not be called")
		},
	}
}

func zoneNilClientCase() support.ResourceCase[*dnsmock.MockedDNS] {
	return support.ResourceCase[*dnsmock.MockedDNS]{
		Name:      "create without a configured dns client",
		Op:        support.OpApply,
		Prepare:   dnsmock.NewUnconfiguredDNS,
		NewConfig: zoneConfig(zoneName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dnsmock.MockedDNS) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "dns api client is null")
			require.Nil(t, state)
		},
	}
}

func TestIntegrationZone_TableDriven(t *testing.T) {
	t.Parallel()

	resource := dnsResource(t, dnssvc.DNSZoneResource)

	cases := []support.ResourceCase[*dnsmock.MockedDNS]{
		zoneCreateCase(),
		zoneCreateTrimsNameCase(),
		zoneCreateAPIFailureCase(),
		zoneCreateReadFailureCase(),
		zoneReadCase(),
		zoneReadAPIFailureCase(),
		zoneRenameRecreatesCase(),
		zoneDeleteCase(),
		zoneDeleteAPIFailureCase(),
		zoneDeleteEmptyNameCase(),
		zoneNilClientCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*dnsmock.MockedDNS])
}

// The case runner cannot build a state with an empty id (ResourceData.State()
// returns nil then), so the name fallback is driven off a raw ResourceData.
func TestIntegrationZoneReadFallsBackToName(t *testing.T) {
	t.Parallel()

	res := dnsResource(t, dnssvc.DNSZoneResource)

	mc := dnsmock.NewMockedDNS()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mc.Client.On("Zone", mock.Anything, zoneName).Return(dnssdk.Zone{Name: zoneRenamed}, nil)

	data := schema.TestResourceDataRaw(t, res.Schema, zoneConfig(zoneName))
	require.Empty(t, data.Id())

	diags := res.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.Equal(t, zoneRenamed, data.Id())
	require.Equal(t, zoneRenamed, data.Get(dnssvc.DNSZoneSchemaName))
}

func TestIntegrationZoneDeleteFallsBackToName(t *testing.T) {
	t.Parallel()

	res := dnsResource(t, dnssvc.DNSZoneResource)

	mc := dnsmock.NewMockedDNS()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mc.Client.On("DeleteZone", mock.Anything, zoneName).Return(nil)

	data := schema.TestResourceDataRaw(t, res.Schema, zoneConfig(zoneName))
	require.Empty(t, data.Id())

	diags := res.DeleteContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	mc.Client.AssertCalled(t, "DeleteZone", mock.Anything, zoneName)
}

// The 255 limit is measured on the raw value, unlike the create path which trims
// first. Everything else about this validator is covered by the co-located
// TestSchemaZoneNameValidation.
func TestIntegrationZoneNameLengthIgnoresPadding(t *testing.T) {
	t.Parallel()

	res := dnsResource(t, dnssvc.DNSZoneResource)

	validate := res.Schema[dnssvc.DNSZoneSchemaName].ValidateDiagFunc
	require.NotNil(t, validate)

	padded := strings.Repeat("a", 250) + strings.Repeat(" ", 10)

	diags := validate(padded, cty.Path{})
	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "dns name can't be empty")

	support.RequireNoDiags(t, validate(strings.TrimSpace(padded), cty.Path{}))
}
