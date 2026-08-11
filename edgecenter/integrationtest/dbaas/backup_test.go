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

func backupConfig(parts ...map[string]interface{}) map[string]interface{} {
	return merge(withCluster(map[string]interface{}{
		"name":        testBackupName,
		"description": "before the migration",
	}), parts...)
}

func sampleBackup(status string) *edgecloud.DBaaSBackup {
	return &edgecloud.DBaaSBackup{
		ID:            testBackupID,
		Name:          testBackupName,
		Description:   "before the migration",
		BackupType:    "MANUAL",
		ClusterID:     testClusterID,
		ParentID:      "",
		Status:        status,
		Size:          12.5,
		IsService:     false,
		HasChild:      false,
		DBMS:          &edgecloud.DBaaSDbmsType{Type: "POSTGRESQL", Version: "17.5"},
		CreatedAt:     "2026-01-01T00:00:00Z",
		UpdatedAt:     "2026-01-02T00:00:00Z",
		FinishedAt:    "2026-01-01T00:10:00Z",
		TaskID:        testTaskID,
		CreatorTaskID: testTaskID,
	}
}

func backupPage(items ...edgecloud.DBaaSBackup) *edgecloud.DBaaSBackupsPage {
	return &edgecloud.DBaaSBackupsPage{Count: len(items), Results: items}
}

func TestIntegrationDBaaSBackupResource_Create(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	var sent edgecloud.DBaaSBackupCreateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "create waits for a finished backup and maps every computed attribute",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupCreate", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(1).(edgecloud.DBaaSBackupCreateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(*sampleBackup(finishedStatus)), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil)

				return mc
			},
			NewConfig: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testBackupID)

				require.Equal(t, testBackupName, sent.Name)
				require.Equal(t, testClusterID, sent.ClusterID)
				require.Equal(t, "before the migration", sent.Description)
				require.Empty(t, sent.ParentID)

				support.RequireStateAttrs(t, state, map[string]string{
					"name":            testBackupName,
					"cluster_id":      testClusterID,
					"backup_type":     "MANUAL",
					"status":          finishedStatus,
					"size":            "12.5",
					"is_service":      "false",
					"has_child":       "false",
					"created_at":      "2026-01-01T00:00:00Z",
					"updated_at":      "2026-01-02T00:00:00Z",
					"finished_at":     "2026-01-01T00:10:00Z",
					"task_id":         testTaskID,
					"creator_task_id": testTaskID,
					"dbms.0.type":     "POSTGRESQL",
					"dbms.0.version":  "17.5",
					"project_id":      testProjectIDStr,
					"region_id":       testRegionIDStr,
				})
			},
		},
		{
			Name: "create forwards the parent id of an incremental backup",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupCreate", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(1).(edgecloud.DBaaSBackupCreateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(*sampleBackup(finishedStatus)), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil)

				return mc
			},
			NewConfig: backupConfig(map[string]interface{}{"parent_id": "parent-backup"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "parent-backup", sent.ParentID)
			},
		},
		{
			Name: "create surfaces the api error and produces no state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupCreate", mock.Anything, mock.Anything).
					Return(nil, statusResponse(409), fmt.Errorf("api error: another backup is running")).Once()

				return mc
			},
			NewConfig: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "another backup is running")
				require.Nil(t, state)
			},
		},
		{
			Name: "a backup that ends in the error status is left orphaned with no state",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupCreate", mock.Anything, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(*sampleBackup("ERROR")), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup("ERROR"), nil, nil).Once()

				return mc
			},
			NewConfig: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "ERROR status")
				require.Nil(t, state, "the backup exists in the api but terraform records nothing")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSBackupResource_Read(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read maps the api answer into state",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":        testBackupName,
					"description": "before the migration",
					"status":      finishedStatus,
					"size":        "12.5",
					"project_id":  testProjectIDStr,
					"region_id":   testRegionIDStr,
				})
			},
		},
		{
			Name: "read clears the dbms block when the api stops returning it",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				backup := sampleBackup(finishedStatus)
				backup.DBMS = nil
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(backup, nil, nil).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"dbms.#": "0"})
			},
		},
		{
			Name: "read drops the resource from state when the backup is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSBackupResource_Update(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	var sent edgecloud.DBaaSBackupUpdateRequest

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "update sends only the changed name",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				renamed := sampleBackup(finishedStatus)
				renamed.Name = "weekly"

				mc.DBaaS.On("BackupUpdate", mock.Anything, testBackupID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(2).(edgecloud.DBaaSBackupUpdateRequest)
					}).
					Return(renamed, nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(renamed, nil, nil).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			NewConfig:    backupConfig(map[string]interface{}{"name": "weekly"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.NotNil(t, sent.Name)
				require.Equal(t, "weekly", *sent.Name)
				require.Nil(t, sent.Description, "an unchanged description must stay out of the request")
				support.RequireStateAttrs(t, state, map[string]string{"name": "weekly"})
			},
		},
		{
			Name: "update surfaces the api error",
			Op:   support.OpApply,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupUpdate", mock.Anything, testBackupID, mock.Anything).
					Return(nil, statusResponse(409), fmt.Errorf("api error: name is taken")).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			NewConfig:    backupConfig(map[string]interface{}{"name": "weekly"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "name is taken")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSBackupResource_Delete(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "delete waits for the task and confirms the backup is gone",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupDelete", mock.Anything, testBackupID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete of an already deleted backup succeeds",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupDelete", mock.Anything, testBackupID).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete fails when the backup is still there after the task",
			Op:   support.OpDelete,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupDelete", mock.Anything, testBackupID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    testBackupID,
			CurrentState: backupConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testBackupID)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationDBaaSBackupResource_Import(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	t.Run("import splits project, region and backup id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(fmt.Sprintf("%d:%d:%s", testProjectID, testRegionID, testBackupID))

		results, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)

		require.Equal(t, testBackupID, results[0].Id())
		require.Equal(t, testProjectID, results[0].Get("project_id"))
		require.Equal(t, testRegionID, results[0].Get("region_id"))
	})

	t.Run("import rejects a malformed id", func(t *testing.T) {
		t.Parallel()

		data := resource.Data(nil)
		data.SetId(testBackupID)

		_, err := resource.Importer.StateContext(t.Context(), data, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "importing DBaaS backup")
	})
}
