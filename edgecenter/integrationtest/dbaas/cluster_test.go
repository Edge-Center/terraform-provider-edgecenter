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

func clusterConfig(parts ...map[string]interface{}) map[string]interface{} {
	base := merge(projectRegion(), map[string]interface{}{
		"name":              testClusterName,
		"description":       "primary postgres",
		"flavor":            "g1-standard-2-4",
		"high_availability": true,
		"dbms": []interface{}{map[string]interface{}{
			"type":    "POSTGRESQL",
			"version": "17.5",
		}},
		"volume": []interface{}{map[string]interface{}{
			"volume_size": 20,
			"volume_type": "db_standard",
		}},
		"interface": []interface{}{map[string]interface{}{
			"network_id": "net-1",
			"subnet_id":  "sub-1",
		}},
	})

	return merge(base, parts...)
}

func sampleCluster(status string) *edgecloud.DBaaSCluster {
	return &edgecloud.DBaaSCluster{
		ID:               testClusterID,
		ProjectID:        testProjectID,
		RegionID:         testRegionID,
		Name:             testClusterName,
		Description:      "primary postgres",
		Status:           status,
		DBMS:             &edgecloud.DBaaSDbmsType{Type: "POSTGRESQL", Version: "17.5"},
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-02T00:00:00Z",
		TaskID:           testTaskID,
		CreatorTaskID:    testTaskID,
		HighAvailability: true,
		Flavor:           "g1-standard-2-4",
		Volume:           &edgecloud.DBaaSVolume{Size: 20, Type: edgecloud.VolumeType("db_standard")},
		Interface:        &edgecloud.DBaaSClusterInterface{NetworkID: "net-1", SubnetID: "sub-1"},
		Connection:       &edgecloud.DBaaSConnection{Method: "direct", Host: "pg.example.com", Port: 5432},
	}
}

func clusterState() map[string]interface{} {
	return clusterConfig()
}

func TestIntegrationDBaaSClusterResource_Create(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	var sent edgecloud.DBaaSClusterCreateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "create sends the whole config, waits for a healthy cluster and stores the id",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(1).(edgecloud.DBaaSClusterCreateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{{ID: testClusterID, Name: testClusterName}}, nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil)

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)

				require.Equal(t, testClusterName, sent.Name)
				require.Equal(t, "primary postgres", sent.Description)
				require.True(t, sent.HighAvailability)
				require.Equal(t, "g1-standard-2-4", sent.Flavor)
				require.Equal(t, edgecloud.DBaaSDbmsType{Type: "POSTGRESQL", Version: "17.5"}, sent.DBMS)
				require.Equal(t, edgecloud.DBaaSVolume{Size: 20, Type: edgecloud.VolumeType("db_standard")}, sent.Volume)
				require.Equal(t, edgecloud.DBaaSClusterInterface{NetworkID: "net-1", SubnetID: "sub-1"}, sent.Interface)

				support.RequireStateAttrs(t, state, map[string]string{
					"name":                   testClusterName,
					"status":                 healthyStatus,
					"flavor":                 "g1-standard-2-4",
					"project_id":             testProjectIDStr,
					"region_id":              testRegionIDStr,
					"dbms.0.type":            "POSTGRESQL",
					"dbms.0.version":         "17.5",
					"volume.0.volume_size":   "20",
					"volume.0.volume_type":   "db_standard",
					"interface.0.network_id": "net-1",
					"interface.0.subnet_id":  "sub-1",
					"connection_info.0.host": "pg.example.com",
					"connection_info.0.port": "5432",
					"task_id":                testTaskID,
					"created_at":             "2026-01-01T00:00:00Z",
					"updated_at":             "2026-01-02T00:00:00Z",
				})
			},
		},
		{
			Name: "create surfaces the api error and produces no state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(nil, statusResponse(500), fmt.Errorf("api error: quota exceeded")).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "quota exceeded")
				require.Nil(t, state)
			},
		},
		{
			Name: "a failed health wait leaves no state although the cluster already exists",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.DBaaS.On("ClustersList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSCluster{{ID: testClusterID, Name: testClusterName}}, nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster("PROVISIONING"), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, statusResponse(500), fmt.Errorf("api error: backend is down")).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				require.Nil(t, state, "the id is stored only after the wait, so the live cluster is orphaned")
			},
		},
		{
			Name: "create refuses an answer without a task id",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(taskResponse(), nil, nil).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "no task IDs")
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSClusterResource_Read(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read maps every nested block back into state",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                   testClusterName,
					"description":            "primary postgres",
					"high_availability":      "true",
					"status":                 healthyStatus,
					"dbms.0.version":         "17.5",
					"volume.0.volume_size":   "20",
					"interface.0.network_id": "net-1",
					"connection_info.0.port": "5432",
				})
			},
		},
		{
			Name: "read drops the resource from state when the cluster is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state, "a cleared id must leave no state")
			},
		},
		{
			Name: "read never clears a task id or a nested block that the api stopped returning",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				cluster := sampleCluster(healthyStatus)
				cluster.TaskID = ""
				cluster.Connection = nil

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(cluster, nil, nil).Once()

				return mc
			},
			CurrentID: testClusterID,
			CurrentState: merge(clusterState(), map[string]interface{}{
				"task_id": testTaskID,
				"connection_info": []interface{}{map[string]interface{}{
					"host": "pg.example.com",
					"port": 5432,
				}},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"task_id":                testTaskID,
					"connection_info.#":      "1",
					"connection_info.0.host": "pg.example.com",
				})
			},
		},
		{
			Name: "read keeps the resource and reports the error on a non 404 failure",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, statusResponse(500), fmt.Errorf("api error: backend is down")).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "backend is down")
				support.RequireStateID(t, state, testClusterID)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSClusterResource_Update(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	var sent edgecloud.DBaaSClusterUpdateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "update sends the changed flavor and volume and always resends the name",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterUpdate", mock.Anything, testClusterID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(2).(edgecloud.DBaaSClusterUpdateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			NewConfig: clusterConfig(map[string]interface{}{
				"flavor": "g1-standard-4-8",
				"volume": []interface{}{map[string]interface{}{
					"volume_size": 40,
					"volume_type": "db_standard",
				}},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)

				require.NotNil(t, sent.Name)
				require.Equal(t, testClusterName, *sent.Name)
				require.Equal(t, "g1-standard-4-8", sent.Flavor)
				require.NotNil(t, sent.Volume)
				require.Equal(t, 40, sent.Volume.Size)
				require.Nil(t, sent.Description, "an unchanged description must stay out of the request")
			},
		},
		{
			Name: "update surfaces the api error",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterUpdate", mock.Anything, testClusterID, mock.Anything).
					Return(nil, statusResponse(500), fmt.Errorf("api error: flavor unavailable")).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"flavor": "g1-standard-4-8"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "flavor unavailable")
				support.RequireStateID(t, state, testClusterID)
			},
		},
		{
			Name: "update reports success although the update task ended in error",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterUpdate", mock.Anything, testClusterID, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"flavor": "g1-standard-4-8"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterID)
				support.RequireStateAttrs(t, state, map[string]string{
					"flavor": "g1-standard-2-4",
				})
			},
		},
		{
			Name: "update returns without waiting when the api answers with no task",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterUpdate", mock.Anything, testClusterID, mock.Anything).
					Return(taskResponse(), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"flavor": "g1-standard-4-8"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				mc.Tasks.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSClusterResource_Delete(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "delete waits for the task and clears the id",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete fails when the api returns no task",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "no tasks returned")
				support.RequireStateID(t, state, testClusterID)
			},
		},
		{
			Name: "delete treats a failed task on an already deleted cluster as done",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "a delete task in the error state surfaces the raw sdk error, never the resource wording",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()
				mc.DBaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(healthyStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testClusterID,
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "task with error state")
				support.RequireStateID(t, state, testClusterID)

				for _, d := range diags {
					require.NotContains(t, d.Summary, "cannot delete DBaaS cluster",
						"WaitAndGetTaskInfo already errors on an error state, so that branch cannot run")
				}
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSClusterResource_CreateAfterAnOrphan(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "a retry after a lost create collides with the name of its own orphan",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()

				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(nil, statusResponse(409),
						fmt.Errorf("cluster with name %s already exists", testClusterName)).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "already exists")
				require.Nil(t, state, "the orphan of the previous attempt now blocks every retry")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSClusterResource_Import(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	t.Run("import splits project, region and cluster id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s", testProjectID, testRegionID, testClusterID))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)

		require.Equal(t, testClusterID, results[0].Id())
		require.Equal(t, testProjectID, results[0].Get("project_id"))
		require.Equal(t, testRegionID, results[0].Get("region_id"))
	})

	t.Run("import rejects a malformed id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId("not-a-triple")

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "importing DBaaS cluster")
	})
}
