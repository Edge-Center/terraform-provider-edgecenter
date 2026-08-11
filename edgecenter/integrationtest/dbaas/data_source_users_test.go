//go:build integration

package dbaas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dbaas/mock"
	dbaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
)

func userList(users ...edgecloud.DBaaSUser) []edgecloud.DBaaSUser {
	return users
}

func userWithDatabases(name string, databases ...string) edgecloud.DBaaSUser {
	granted := make([]edgecloud.DBaaSUserDatabase, 0, len(databases))
	for _, database := range databases {
		granted = append(granted, edgecloud.DBaaSUserDatabase{Name: database})
	}

	return edgecloud.DBaaSUser{Name: name, Databases: granted}
}

func TestIntegrationDBaaSUsersDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := dbaasDataSource(t, dbaassvc.DBaaSUsersDataSource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read without a filter returns every user and its grants",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UsersList", mock.Anything, testClusterID, mock.Anything).
					Return(userList(
						userWithDatabases(testUsername, "appdb", "reports"),
						userWithDatabases("readonly"),
					), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(nil),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
				support.RequireStateAttrs(t, state, map[string]string{
					"items.#":             "2",
					"items.0.name":        testUsername,
					"items.0.databases.#": "2",
					"items.0.databases.0": "appdb",
					"items.0.databases.1": "reports",
					"items.1.name":        "readonly",
					"items.1.databases.#": "0",
				})
			},
		},
		{
			Name: "read with a name filter keeps only the matching user",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UsersList", mock.Anything, testClusterID, mock.Anything).
					Return(userList(
						userWithDatabases(testUsername, "appdb"),
						userWithDatabases("readonly"),
					), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(map[string]interface{}{"name": "readonly"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"items.#":      "1",
					"items.0.name": "readonly",
				})
			},
		},
		{
			Name: "read fails instead of returning an empty list when the filter matches nothing",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UsersList", mock.Anything, testClusterID, mock.Anything).
					Return(userList(userWithDatabases(testUsername)), nil, nil).Once()

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
			Name: "read surfaces the api error",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UsersList", mock.Anything, testClusterID, mock.Anything).
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
			Name: "the users and the databases data source of one cluster share an id",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UsersList", mock.Anything, testClusterID, mock.Anything).
					Return(userList(userWithDatabases(testUsername)), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: withCluster(map[string]interface{}{"name": testUsername}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
