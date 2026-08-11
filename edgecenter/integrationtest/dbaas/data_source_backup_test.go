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

const backupPageSize = 100

func backupNamed(id, name string) edgecloud.DBaaSBackup {
	backup := sampleBackup(finishedStatus)
	backup.ID = id
	backup.Name = name

	return *backup
}

func fullBackupPage(total int, name string) *edgecloud.DBaaSBackupsPage {
	items := make([]edgecloud.DBaaSBackup, 0, backupPageSize)
	for i := 0; i < backupPageSize; i++ {
		items = append(items, backupNamed(fmt.Sprintf("filler-%d", i), name+"-filler"))
	}

	return &edgecloud.DBaaSBackupsPage{Count: total, Results: items}
}

func TestIntegrationDBaaSBackupDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := dbaasDataSource(t, dbaassvc.DBaaSBackupDataSource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "lookup by id fetches the backup directly",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testBackupID}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testBackupID)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":            testBackupName,
					"description":     "before the migration",
					"cluster_id":      testClusterID,
					"backup_type":     "MANUAL",
					"status":          finishedStatus,
					"size":            "12.5",
					"is_service":      "false",
					"has_child":       "false",
					"created_at":      "2026-01-01T00:00:00Z",
					"finished_at":     "2026-01-01T00:10:00Z",
					"creator_task_id": testTaskID,
					"dbms.0.type":     "POSTGRESQL",
					"project_id":      testProjectIDStr,
					"region_id":       testRegionIDStr,
				})
				mc.DBaaS.AssertNotCalled(t, "BackupsListPage", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "lookup by name searches the backup list and then fetches by id",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(
						backupNamed("other-id", "weekly"),
						backupNamed(testBackupID, testBackupName),
					), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testBackupName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testBackupID)

				for _, call := range mc.DBaaS.Calls {
					if call.Method == "BackupsListPage" {
						opts := call.Arguments.Get(1).(*edgecloud.DBaaSBackupListOptions)
						require.Equal(t, testBackupName, opts.Search)
						require.Equal(t, backupPageSize, opts.Limit)
					}
				}
			},
		},
		{
			Name: "lookup by name walks every page of the search result",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.MatchedBy(func(opts *edgecloud.DBaaSBackupListOptions) bool {
					return opts.Offset == 0
				})).Return(fullBackupPage(backupPageSize+1, testBackupName), nil, nil).Once()
				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.MatchedBy(func(opts *edgecloud.DBaaSBackupListOptions) bool {
					return opts.Offset == backupPageSize
				})).Return(backupPage(backupNamed(testBackupID, testBackupName)), nil, nil).Once()
				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(sampleBackup(finishedStatus), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testBackupName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testBackupID)
			},
		},
		{
			Name: "lookup by name fails when the search returns no exact match",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(backupNamed("other-id", testBackupName+"-old")), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testBackupName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "was not found")
			},
		},
		{
			Name: "lookup by name refuses an ambiguous answer",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupsListPage", mock.Anything, mock.Anything).
					Return(backupPage(
						backupNamed(testBackupID, testBackupName),
						backupNamed("other-id", testBackupName),
					), nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"name": testBackupName}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "specify 'id' instead")
			},
		},
		{
			Name: "lookup by id fails when the backup is gone",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(nil, notFound(), fmt.Errorf("not found")).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testBackupID}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "was not found")
			},
		},
		{
			Name: "lookup by id refuses an empty api answer",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("BackupGet", mock.Anything, testBackupID, false).
					Return(nil, statusResponse(200), nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"id": testBackupID}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "empty API response")
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
