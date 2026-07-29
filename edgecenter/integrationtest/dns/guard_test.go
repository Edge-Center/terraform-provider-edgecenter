//go:build integration

package dns_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dnsmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dns/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dns"
)

const (
	guardDiagSummary = "dns api client is null. make sure that you defined edgecenter_dns_api var in edgecenter provider section."
	guardNamePrefix  = "edgecenter_dns_"
	guardSeededID    = "guard.example.com"
)

var (
	guardAPIError = errors.New("dns api unavailable")

	guardResourceNames = []string{
		dns.DNSZoneResource,
		dns.DNSZoneRecordResource,
		dns.DNSSecondaryZoneResource,
	}

	guardDataSourceNames = []string{
		dns.DNSSecondaryZonesDataSource,
	}
)

type guardContext struct {
	op string
	fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics
}

type guardTarget struct {
	name     string
	resource *schema.Resource
	contexts []guardContext
}

func guardContexts(res *schema.Resource) []guardContext {
	contexts := make([]guardContext, 0, 4)

	if res.CreateContext != nil {
		contexts = append(contexts, guardContext{op: "CreateContext", fn: res.CreateContext})
	}
	if res.ReadContext != nil {
		contexts = append(contexts, guardContext{op: "ReadContext", fn: res.ReadContext})
	}
	if res.UpdateContext != nil {
		contexts = append(contexts, guardContext{op: "UpdateContext", fn: res.UpdateContext})
	}
	if res.DeleteContext != nil {
		contexts = append(contexts, guardContext{op: "DeleteContext", fn: res.DeleteContext})
	}

	return contexts
}

func guardNames(registry map[string]*schema.Resource) []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		if strings.HasPrefix(name, guardNamePrefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	return names
}

func guardTargets(t *testing.T) []guardTarget {
	t.Helper()

	p := provider.Provider()

	resourceNames := guardNames(p.ResourcesMap)
	dataSourceNames := guardNames(p.DataSourcesMap)

	require.ElementsMatch(t, guardResourceNames, resourceNames, "registered %s* resources", guardNamePrefix)
	require.ElementsMatch(t, guardDataSourceNames, dataSourceNames, "registered %s* data sources", guardNamePrefix)

	targets := make([]guardTarget, 0, len(resourceNames)+len(dataSourceNames))
	for _, name := range resourceNames {
		res := p.ResourcesMap[name]
		targets = append(targets, guardTarget{name: name, resource: res, contexts: guardContexts(res)})
	}
	for _, name := range dataSourceNames {
		ds := p.DataSourcesMap[name]
		targets = append(targets, guardTarget{name: name, resource: ds, contexts: guardContexts(ds)})
	}

	for _, target := range targets {
		require.NotEmpty(t, target.contexts, "%s exposes no context functions", target.name)
	}

	return targets
}

func guardInvoke(t *testing.T, target guardTarget, gc guardContext, meta interface{}) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()

	data := support.NewResourceDataFromState(t, target.resource, support.NewState(t, target.resource, nil, guardSeededID))

	return data, gc.fn(context.Background(), data, meta)
}

func guardMockedDNS() *dnsmock.MockedDNS {
	mc := dnsmock.NewMockedDNS()

	mc.Client.On("CreateZone", mock.Anything, mock.Anything).
		Return(uint64(0), guardAPIError).Maybe()
	mc.Client.On("Zone", mock.Anything, mock.Anything).
		Return(dnssdk.Zone{}, guardAPIError).Maybe()
	mc.Client.On("DeleteZone", mock.Anything, mock.Anything).
		Return(guardAPIError).Maybe()
	mc.Client.On("RRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(dnssdk.RRSet{}, guardAPIError).Maybe()
	mc.Client.On("CreateRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(guardAPIError).Maybe()
	mc.Client.On("UpdateRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(guardAPIError).Maybe()
	mc.Client.On("DeleteRRSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(guardAPIError).Maybe()
	mc.Client.On("CreateSecondaryZone", mock.Anything, mock.Anything).
		Return(dnssdk.SecondaryZone{}, guardAPIError).Maybe()
	mc.Client.On("GetSecondaryZone", mock.Anything, mock.Anything).
		Return(dnssdk.SecondaryZone{}, guardAPIError).Maybe()
	mc.Client.On("UpdateSecondaryZone", mock.Anything, mock.Anything, mock.Anything).
		Return(dnssdk.SecondaryZone{}, guardAPIError).Maybe()
	mc.Client.On("DeleteSecondaryZone", mock.Anything, mock.Anything).
		Return(guardAPIError).Maybe()
	mc.Client.On("SecondaryZones", mock.Anything).
		Return([]dnssdk.SecondaryZone(nil), guardAPIError).Maybe()

	return mc
}

func TestIntegrationDNSDependencyNilClient(t *testing.T) {
	t.Parallel()

	for _, target := range guardTargets(t) {
		target := target
		for _, gc := range target.contexts {
			gc := gc
			t.Run(target.name+"/"+gc.op, func(t *testing.T) {
				t.Parallel()

				mc := dnsmock.NewUnconfiguredDNS()

				data, diags := guardInvoke(t, target, gc, mc.TestMeta())

				support.RequireOnlyErrorDiags(t, diags)
				require.Len(t, diags, 1)
				require.Equal(t, guardDiagSummary, diags[0].Summary)
				require.Equal(t, guardSeededID, data.Id(), "guarded call must not reach the crud function")
			})
		}
	}
}

func TestIntegrationDNSDependencyPassesThrough(t *testing.T) {
	t.Parallel()

	for _, target := range guardTargets(t) {
		target := target
		for _, gc := range target.contexts {
			gc := gc
			t.Run(target.name+"/"+gc.op, func(t *testing.T) {
				t.Parallel()

				mc := guardMockedDNS()
				t.Cleanup(func() { mc.MockCleanup(t) })

				_, diags := guardInvoke(t, target, gc, mc.TestMeta())

				for _, d := range diags {
					require.NotEqual(t, guardDiagSummary, d.Summary, "guard must not fire with a configured client")
				}
				support.RequireErrorDiagContains(t, diags, guardAPIError.Error())
			})
		}
	}
}
