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

func userConfig(databases ...string) map[string]interface{} {
	names := make([]interface{}, 0, len(databases))
	for _, name := range databases {
		names = append(names, name)
	}

	return withCluster(map[string]interface{}{
		"name":      testUsername,
		"password":  testPassword,
		"databases": names,
	})
}

func sampleUser(databases ...string) *edgecloud.DBaaSUser {
	granted := make([]edgecloud.DBaaSUserDatabase, 0, len(databases))
	for _, name := range databases {
		granted = append(granted, edgecloud.DBaaSUserDatabase{Name: name})
	}

	return &edgecloud.DBaaSUser{Name: testUsername, Databases: granted}
}

func TestIntegrationDBaaSUserResource_Create(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	var sent edgecloud.DBaaSUserCreateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "create sends name, password and every requested database",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserCreate", mock.Anything, testClusterID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(2).(edgecloud.DBaaSUserCreateRequest)
					}).
					Return(sampleUser("appdb", "reports"), nil, nil).Once()
				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("appdb", "reports"), nil, nil).Once()

				return mc
			},
			NewConfig: userConfig("appdb", "reports"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testUsername)

				require.Equal(t, testUsername, sent.Name)
				require.Equal(t, testPassword, sent.Password)
				require.Equal(t, []edgecloud.DBaaSUserDatabase{{Name: "appdb"}, {Name: "reports"}}, sent.Databases)

				support.RequireStateAttrs(t, state, map[string]string{
					"name":        testUsername,
					"password":    testPassword,
					"cluster_id":  testClusterID,
					"databases.#": "2",
					"databases.0": "appdb",
					"databases.1": "reports",
				})
			},
		},
		{
			Name: "create sends no database list when the config grants none",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserCreate", mock.Anything, testClusterID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(2).(edgecloud.DBaaSUserCreateRequest)
					}).
					Return(sampleUser(), nil, nil).Once()
				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser(), nil, nil).Once()

				return mc
			},
			NewConfig: userConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Empty(t, sent.Databases)
			},
		},
		{
			Name: "create surfaces the api error and produces no state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserCreate", mock.Anything, testClusterID, mock.Anything).
					Return(nil, statusResponse(400), fmt.Errorf("api error: password too short")).Once()

				return mc
			},
			NewConfig: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "password too short")
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSUserResource_Read(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read replaces the database list with what the api reports",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("reports"), nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb", "reports"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"databases.#": "1",
					"databases.0": "reports",
				})
			},
		},
		{
			Name: "read never refreshes the password, so a password changed outside terraform stays invisible",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("appdb"), nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"password": testPassword})
			},
		},
		{
			Name: "read drops the resource from state when the user is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read keeps the resource and reports the error on a non 404 failure",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(nil, statusResponse(500), fmt.Errorf("api error: backend is down")).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "backend is down")
				support.RequireStateID(t, state, testUsername)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSUserResource_Update(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	var sent edgecloud.DBaaSUserUpdateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "update sends the new password and leaves the grants alone",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserUpdate", mock.Anything, testClusterID, testUsername, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(3).(edgecloud.DBaaSUserUpdateRequest)
					}).
					Return(nil, nil).Once()
				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("appdb"), nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			NewConfig: withCluster(map[string]interface{}{
				"name":      testUsername,
				"password":  "new-password",
				"databases": []interface{}{"appdb"},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "new-password", sent.Password)
				mc.DBaaS.AssertNotCalled(t, "UserGrantAccess", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				mc.DBaaS.AssertNotCalled(t, "UserRevokeAccess", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			Name: "update grants the added database and revokes the removed one",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGrantAccess", mock.Anything, testClusterID, testUsername, "reports").
					Return(nil, nil).Once()
				mc.DBaaS.On("UserRevokeAccess", mock.Anything, testClusterID, testUsername, "appdb").
					Return(nil, nil).Once()
				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("reports"), nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			NewConfig:    userConfig("reports"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				mc.DBaaS.AssertNotCalled(t, "UserUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				support.RequireStateAttrs(t, state, map[string]string{
					"databases.#": "1",
					"databases.0": "reports",
				})
			},
		},
		{
			Name: "a failed password update still records the new password in state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserUpdate", mock.Anything, testClusterID, testUsername, mock.Anything).
					Return(statusResponse(400), fmt.Errorf("api error: password policy")).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			NewConfig: withCluster(map[string]interface{}{
				"name":      testUsername,
				"password":  "new-password",
				"databases": []interface{}{"appdb"},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "password policy")
				support.RequireStateAttrs(t, state, map[string]string{"password": "new-password"})
			},
		},
		{
			Name: "reordering the grant list issues no grant, revoke or password call",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGet", mock.Anything, testClusterID, testUsername).
					Return(sampleUser("appdb", "reports"), nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb", "reports"),
			NewConfig:    userConfig("reports", "appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				mc.DBaaS.AssertNotCalled(t, "UserGrantAccess", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				mc.DBaaS.AssertNotCalled(t, "UserRevokeAccess", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				mc.DBaaS.AssertNotCalled(t, "UserUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				support.RequireStateAttrs(t, state, map[string]string{
					"databases.0": "appdb",
					"databases.1": "reports",
				})
			},
		},
		{
			Name: "update surfaces a failed grant",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGrantAccess", mock.Anything, testClusterID, testUsername, "reports").
					Return(statusResponse(403), fmt.Errorf("api error: grant refused")).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			NewConfig:    userConfig("appdb", "reports"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "grant refused")
			},
		},
		{
			Name: "update writes the full desired list into state although a grant failed",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserGrantAccess", mock.Anything, testClusterID, testUsername, "reports").
					Return(nil, nil).Maybe()
				mc.DBaaS.On("UserGrantAccess", mock.Anything, testClusterID, testUsername, "audit").
					Return(statusResponse(403), fmt.Errorf("api error: grant refused")).Maybe()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			NewConfig:    userConfig("appdb", "reports", "audit"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"databases.#": "3",
					"databases.0": "appdb",
					"databases.1": "reports",
					"databases.2": "audit",
				})
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSUserResource_Delete(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "delete removes the user and clears the id",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserDelete", mock.Anything, testClusterID, testUsername).
					Return(nil, nil).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete of an already deleted user fails and keeps the resource in state",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("UserDelete", mock.Anything, testClusterID, testUsername).
					Return(notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testUsername,
			CurrentState: userConfig("appdb"),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testUsername)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSUserResource_Import(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	t.Run("import splits project, region, cluster and username", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s:%s", testProjectID, testRegionID, testClusterID, testUsername))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)

		require.Equal(t, testUsername, results[0].Id())
		require.Equal(t, testProjectID, results[0].Get("project_id"))
		require.Equal(t, testRegionID, results[0].Get("region_id"))
		require.Equal(t, testClusterID, results[0].Get("cluster_id"))
	})

	t.Run("import leaves the required password unset", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s:%s", testProjectID, testRegionID, testClusterID, testUsername))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)

		require.Empty(t, results[0].Get("password"))
	})

	t.Run("import rejects a malformed id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(testUsername)

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "importing DBaaS user")
	})
}
