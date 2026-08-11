//go:build integration

package dbaas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dbaas/mock"
	dbaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
)

func TestIntegrationDBaaSDatabasesDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := dbaasDataSource(t, dbaassvc.DBaaSDatabasesDataSource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read without a filter returns every database of the cluster",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("appdb", "reports"), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(nil),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
				support.RequireStateAttrs(t, state, map[string]string{
					"items.#":      "2",
					"items.0.name": "appdb",
					"items.1.name": "reports",
				})
			},
		},
		{
			Name: "read with a name filter keeps only the matching database",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("appdb", "reports"), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(map[string]interface{}{"name": "reports"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"items.#":      "1",
					"items.0.name": "reports",
				})
			},
		},
		{
			Name: "read fails instead of returning an empty list when the filter matches nothing",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("appdb"), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(map[string]interface{}{"name": "missing"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "not found in cluster")
			},
		},
		{
			Name: "read of an empty cluster without a filter succeeds with no items",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList(), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(nil),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"items.#": "0"})
			},
		},
		{
			Name: "read surfaces the api error",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(nil, statusResponse(500), fmt.Errorf("api error: backend is down")).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(nil),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "backend is down")
			},
		},
		{
			Name: "read sends no paging options to the database list",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("appdb"), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(nil),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)

				for _, call := range mc.DBaaS.Calls {
					if call.Method == "DatabasesList" {
						require.Nil(t, call.Arguments.Get(2))
					}
				}
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
