//go:build integration

package dbaas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dbaas/mock"
	dbaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
)

func clustersDataSourceState(parts map[string]interface{}) map[string]interface{} {
	return merge(projectRegion(), parts)
}

func TestIntegrationDBaaSClustersDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := dbaasDataSource(t, dbaassvc.DBaaSClustersDataSource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "lookup by id fetches the cluster directly",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"id": testClusterID}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                   testClusterName,
					"description":            "primary postgres",
					"status":                 healthyStatus,
					"flavor":                 "g1-standard-2-4",
					"high_availability":      "true",
					"dbms.0.type":            "POSTGRESQL",
					"volume.0.volume_size":   "20",
					"interface.0.subnet_id":  "sub-1",
					"connection_info.0.host": "pg.example.com",
					"task_id":                testTaskID,
				})
				mc.DBaaS.AssertNotCalled(t, "ClustersList", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "lookup by name resolves through the cluster list",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{
						{ID: "other-id", Name: "pg-spare"},
						{ID: testClusterID, Name: testClusterName},
					}, nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
			},
		},
		{
			Name: "lookup by name fails when no cluster carries it",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{{ID: "other-id", Name: "pg-spare"}}, nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "not found")
			},
		},
		{
			Name: "lookup by name refuses an ambiguous answer",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{
						{ID: testClusterID, Name: testClusterName},
						{ID: "other-id", Name: testClusterName},
					}, nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "multiple DBaaS clusters")
			},
		},
		{
			Name: "lookup by name sends no paging options to the cluster list",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{{ID: testClusterID, Name: testClusterName}}, nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"name": testClusterName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)

				for _, call := range mc.DBaaS.Calls {
					if call.Method == "ClustersList" {
						require.Nil(t, call.Arguments.Get(1))
					}
				}
			},
		},
		{
			Name: "lookup by id fails when the cluster is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: clustersDataSourceState(map[string]interface{}{"id": testClusterID}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "not found")
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
