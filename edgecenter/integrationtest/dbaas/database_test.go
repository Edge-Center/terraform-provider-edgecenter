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

func databaseConfig(parts ...map[string]interface{}) map[string]interface{} {
	return merge(withCluster(map[string]interface{}{
		"name":     testDatabaseName,
		"encoding": "UTF8",
		"locale":   "en_US.UTF-8",
	}), parts...)
}

func databaseList(names ...string) []edgecloud.DBaaSDatabase {
	items := make([]edgecloud.DBaaSDatabase, 0, len(names))
	for _, name := range names {
		items = append(items, edgecloud.DBaaSDatabase{Name: name})
	}

	return items
}

func TestIntegrationDBaaSDatabaseResource_Create(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSDatabaseResource)

	var (
		sentCluster string
		sent        edgecloud.DBaaSDatabaseCreateRequest
	)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "create sends name, encoding and locale and stores the name as the id",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseCreate", mock.Anything, testClusterID, mock.Anything).
					Run(func(args mock.Arguments) {
						sentCluster = args.Get(1).(string)
						sent = args.Get(2).(edgecloud.DBaaSDatabaseCreateRequest)
					}).
					Return(&edgecloud.DBaaSDatabase{Name: testDatabaseName}, nil, nil).Once()
				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList(testDatabaseName), nil, nil).Once()

				return mc
			},
			NewConfig: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testDatabaseName)

				require.Equal(t, testClusterID, sentCluster)
				require.Equal(t, testDatabaseName, sent.Name)
				require.Equal(t, "UTF8", sent.Encoding)
				require.Equal(t, "en_US.UTF-8", sent.Locale)
			},
		},
		{
			Name: "create omits encoding and locale when the config leaves them out",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseCreate", mock.Anything, testClusterID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(2).(edgecloud.DBaaSDatabaseCreateRequest)
					}).
					Return(&edgecloud.DBaaSDatabase{Name: testDatabaseName}, nil, nil).Once()
				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList(testDatabaseName), nil, nil).Once()

				return mc
			},
			NewConfig: withCluster(map[string]interface{}{"name": testDatabaseName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Empty(t, sent.Encoding)
				require.Empty(t, sent.Locale)
			},
		},
		{
			Name: "create surfaces the api error and produces no state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseCreate", mock.Anything, testClusterID, mock.Anything).
					Return(nil, statusResponse(409), fmt.Errorf("api error: database already exists")).Once()

				return mc
			},
			NewConfig: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "database already exists")
				require.Nil(t, state)
			},
		},
		{
			Name: "a database the api does not list right after create is dropped from state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseCreate", mock.Anything, testClusterID, mock.Anything).
					Return(&edgecloud.DBaaSDatabase{Name: testDatabaseName}, nil, nil).Once()
				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList(), nil, nil).Once()

				return mc
			},
			NewConfig: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state, "the id is cleared, so terraform records a created database as absent")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSDatabaseResource_Read(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSDatabaseResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read keeps the resource when the cluster still lists the database",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("other", testDatabaseName), nil, nil).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testDatabaseName)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":       testDatabaseName,
					"cluster_id": testClusterID,
				})
			},
		},
		{
			Name: "read drops the resource from state when the database is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList("other"), nil, nil).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read of a database whose cluster is gone fails the plan instead of clearing the id",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(nil, notFound(), fmt.Errorf("cluster not found")).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "cluster not found")
				support.RequireStateID(t, state, testDatabaseName)
			},
		},
		{
			Name: "read leaves encoding and locale at their state values because the api never returns them",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabasesList", mock.Anything, testClusterID, mock.Anything).
					Return(databaseList(testDatabaseName), nil, nil).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"encoding": "UTF8",
					"locale":   "en_US.UTF-8",
				})
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSDatabaseResource_Delete(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSDatabaseResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "delete removes the database and clears the id",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseDelete", mock.Anything, testClusterID, testDatabaseName).
					Return(&edgecloud.DBaaSDatabase{Name: testDatabaseName}, nil, nil).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete of a database whose cluster is gone fails, so even -refresh=false cannot destroy it",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseDelete", mock.Anything, testClusterID, testDatabaseName).
					Return(nil, notFound(), fmt.Errorf("cluster not found")).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "cluster not found")
				support.RequireStateID(t, state, testDatabaseName)
			},
		},
		{
			Name: "delete of an already deleted database fails and keeps the resource in state",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DatabaseDelete", mock.Anything, testClusterID, testDatabaseName).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testDatabaseName,
			CurrentState: databaseConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testDatabaseName)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSDatabaseResource_Import(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSDatabaseResource)

	t.Run("import splits project, region, cluster and database name", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s:%s", testProjectID, testRegionID, testClusterID, testDatabaseName))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)

		require.Equal(t, testDatabaseName, results[0].Id())
		require.Equal(t, testProjectID, results[0].Get("project_id"))
		require.Equal(t, testRegionID, results[0].Get("region_id"))
		require.Equal(t, testClusterID, results[0].Get("cluster_id"))
	})

	t.Run("import leaves encoding and locale unset", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s:%s", testProjectID, testRegionID, testClusterID, testDatabaseName))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)

		require.Empty(t, results[0].Get("encoding"))
		require.Empty(t, results[0].Get("locale"))
	})

	t.Run("import rejects a malformed id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s", testProjectID, testRegionID, testClusterID))

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "importing DBaaS database")
	})

	t.Run("import rejects the very id that create publishes", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(testDatabaseName)

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "wrong input id")
	})
}
