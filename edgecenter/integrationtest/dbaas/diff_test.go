//go:build integration

package dbaas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
)

func planDiff(
	t *testing.T,
	resource *schema.Resource,
	current *terraform.InstanceState,
	config map[string]interface{},
) *terraform.InstanceDiff {
	t.Helper()

	diff, err := resource.Diff(t.Context(), current, terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)

	return diff
}

func importedState(t *testing.T, resource *schema.Resource, importID string) *terraform.InstanceState {
	t.Helper()

	data := resource.Data(nil)
	data.SetId(importID)

	results, err := resource.Importer.StateContext(t.Context(), data, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)

	return results[0].State()
}

func importedDatabaseState(t *testing.T, resource *schema.Resource, importID string) *terraform.InstanceState {
	t.Helper()

	state := importedState(t, resource, importID)
	state.Attributes["name"] = testDatabaseName

	return state
}

func namedScope() map[string]interface{} {
	return map[string]interface{}{
		"project_name": "test-project",
		"region_name":  "test-region",
	}
}

func TestIntegrationDBaaSBackupResource_Diff(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSBackupResource)

	t.Run("a name based config plans a replacement because read stored the resolved ids", func(t *testing.T) {
		t.Parallel()

		config := merge(namedScope(), map[string]interface{}{
			"name":        testBackupName,
			"description": "before the migration",
			"cluster_id":  testClusterID,
		})
		afterRead := merge(config, map[string]interface{}{
			"project_id": testProjectID,
			"region_id":  testRegionID,
		})

		diff := planDiff(t, resource, support.NewState(t, resource, afterRead, testBackupID), config)

		require.False(t, diff.Empty(), "the resolved ids are not in config, so they diff back to zero")
		require.True(t, diff.RequiresNew(), "both ids are force new, so every apply destroys the backup")
	})

	t.Run("an id based config does not plan a replacement after read", func(t *testing.T) {
		t.Parallel()

		config := backupConfig()
		diff := planDiff(t, resource, support.NewState(t, resource, config, testBackupID), config)

		require.False(t, diff.RequiresNew())
	})
}

func TestIntegrationDBaaSClusterResource_Diff(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSClusterResource)

	t.Run("a name based config plans an endless in place update because read stored the resolved ids", func(t *testing.T) {
		t.Parallel()

		config := merge(clusterConfig(), namedScope())
		delete(config, "project_id")
		delete(config, "region_id")

		afterRead := merge(config, map[string]interface{}{
			"project_id": testProjectID,
			"region_id":  testRegionID,
		})

		diff := planDiff(t, resource, support.NewState(t, resource, afterRead, testClusterID), config)

		require.False(t, diff.Empty())
		require.False(t, diff.RequiresNew(), "the cluster does not mark the scope force new, so this is an in place update")
	})

	t.Run("moving the cluster to another region plans an in place update instead of a replacement", func(t *testing.T) {
		t.Parallel()

		diff := planDiff(t,
			resource,
			support.NewState(t, resource, clusterConfig(), testClusterID),
			clusterConfig(map[string]interface{}{"region_id": 9}),
		)

		require.False(t, diff.Empty())
		require.False(t, diff.RequiresNew(), "the update then patches a region that cannot hold the cluster")
	})
}

func TestIntegrationDBaaSDatabaseResource_Diff(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSDatabaseResource)

	importID := fmt.Sprintf("%d:%d:%s:%s", testProjectID, testRegionID, testClusterID, testDatabaseName)

	t.Run("an imported database plans a replacement because encoding and locale are never restored", func(t *testing.T) {
		t.Parallel()

		config := merge(withCluster(map[string]interface{}{"name": testDatabaseName}), map[string]interface{}{
			"encoding": "UTF8",
			"locale":   "en_US.UTF-8",
		})

		diff := planDiff(t, resource, importedDatabaseState(t, resource, importID), config)

		require.True(t, diff.RequiresNew(), "the first apply after import drops a live database")
	})

	t.Run("an imported database plans a replacement for a name based config", func(t *testing.T) {
		t.Parallel()

		config := merge(namedScope(), map[string]interface{}{
			"cluster_id": testClusterID,
			"name":       testDatabaseName,
		})

		diff := planDiff(t, resource, importedDatabaseState(t, resource, importID), config)

		require.True(t, diff.RequiresNew(), "the importer writes force new ids the config does not carry")
	})

	t.Run("an imported database without optional attributes does not plan a replacement", func(t *testing.T) {
		t.Parallel()

		config := withCluster(map[string]interface{}{"name": testDatabaseName})
		diff := planDiff(t, resource, importedDatabaseState(t, resource, importID), config)

		require.False(t, diff.RequiresNew())
	})
}

func TestIntegrationDBaaSUserResource_Diff(t *testing.T) {
	t.Parallel()

	resource := dbaasResource(t, dbaassvc.DBaaSUserResource)

	t.Run("reordering the grant list is a change because databases is an ordered list", func(t *testing.T) {
		t.Parallel()

		diff := planDiff(t,
			resource,
			support.NewState(t, resource, userConfig("appdb", "reports"), testUsername),
			userConfig("reports", "appdb"),
		)

		require.False(t, diff.Empty(), "the api collection is unordered, so this diff can never converge")
		require.False(t, diff.RequiresNew())
	})
}
