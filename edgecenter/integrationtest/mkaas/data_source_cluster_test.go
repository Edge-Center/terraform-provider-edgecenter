//go:build integration

package mkaas_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	mkaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/mkaas/mock"
	mkaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/mkaas"
)

func byName(name string) *edgecloud.MKaaSClusterListOptions {
	return &edgecloud.MKaaSClusterListOptions{Name: name, Limit: 2}
}

func TestIntegrationMKaaSClusterDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := mkaasDataSource(t, mkaassvc.MKaaSClusterDataSource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "a lookup by id fills every attribute from the cluster payload",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(nil), nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testClusterIDStr}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                    testClusterName,
					"ssh_keypair_name":        testKeypairName,
					"network_id":              testNetworkID,
					"subnet_id":               testSubnetID,
					"internal_ip":             "10.0.0.10",
					"external_ip":             "203.0.113.10",
					"created":                 "2026-01-01T00:00:00Z",
					"processing":              "false",
					"status":                  "Provisioned",
					"stage":                   "Ready",
					"control_plane.0.flavor":  testFlavor,
					"control_plane.0.version": testShortVersion,
				})
			},
		},
		{
			Name:      "a lookup by id overwrites the configured scope with the payload values",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				cluster := sampleCluster(nil)
				cluster.ProjectID = 999
				cluster.RegionID = 111

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(cluster, nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testClusterIDStr}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"project_id": "999",
					"region_id":  "111",
				})
			},
		},
		{
			Name:      "a lookup by id turns a 404 into a not found message",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), errors.New("cluster not found")).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testClusterIDStr}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "MKaaS cluster 42 not found")
			},
		},
		{
			Name:      "a non numeric id is rejected after the client is built",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": "not-a-number"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "invalid id")
				mc.MKaaS.AssertNotCalled(t, "ClusterGet", mock.Anything, mock.Anything)
			},
		},
		{
			Name:      "a lookup by name asks the list endpoint for two results",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClustersList", mock.Anything, byName(testClusterName)).
					Return([]edgecloud.MKaaSCluster{*sampleCluster(nil)}, nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"id":   testClusterIDStr,
					"name": testClusterName,
				})
			},
		},
		{
			Name:      "a lookup by name accepts a cluster whose name differs from the requested one",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClustersList", mock.Anything, byName(testClusterName)).
					Return([]edgecloud.MKaaSCluster{
						*sampleCluster(map[string]interface{}{"name": testClusterName + "-staging"}),
					}, nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, testClusterName+"-staging", state.Attributes["name"],
					"the read never compares the returned name with the requested one")
			},
		},
		{
			Name:      "a lookup by name fails when the list is empty",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClustersList", mock.Anything, byName(testClusterName)).
					Return([]edgecloud.MKaaSCluster{}, nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags,
					"MKaaS cluster with name "+testClusterName+" not found")
			},
		},
		{
			Name:      "a lookup by name refuses an ambiguous answer",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClustersList", mock.Anything, byName(testClusterName)).
					Return([]edgecloud.MKaaSCluster{*sampleCluster(nil), *sampleCluster(nil)}, nil, nil).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "multiple MKaaS clusters found with name")
			},
		},
		{
			Name:      "a lookup by name surfaces a list error",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClustersList", mock.Anything, byName(testClusterName)).
					Return(nil, statusResponse(500), errors.New("internal error")).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "internal error")
			},
		},
		{
			Name:      "a read without id and without name is rejected",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				return mc
			},
			CurrentState: projectRegion(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "either 'id' or 'name' must be specified")
				mc.MKaaS.AssertNotCalled(t, "ClustersList", mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
