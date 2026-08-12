//go:build integration

package dbaas_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dbaas/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testProjectID    = 1
	testProjectIDStr = "1"
	testRegionID     = 8
	testRegionIDStr  = "8"

	testClusterID   = "0f1b6a2c-9d3e-4a55-8b77-2c2c2c2c2c2c"
	testClusterName = "pg-main"

	testTaskID      = "6e3a1b0e-5f2d-4c8a-9a1b-3d3d3d3d3d3d"
	testOtherTaskID = "9c9c9c9c-1111-2222-3333-444444444444"

	testBackupID   = "b1b1b1b1-2222-3333-4444-555555555555"
	testBackupName = "nightly"

	testDatabaseName = "appdb"
	testUsername     = "appuser"
	testPassword     = "s3cret-pass"

	unsetDataSourceID = "-"

	healthyStatus  = "HEALTHY"
	finishedStatus = "FINISHED"
)

func dbaasResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func dbaasDataSource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().DataSourcesMap[name]
	if res == nil {
		t.Fatalf("data source %q is not registered in the provider", name)
	}

	return res
}

func runDataSourceRead(
	t *testing.T,
	resource *schema.Resource,
	tc support.ResourceCase[*dbaasmock.MockedDBaaS],
	fake *dbaasmock.MockedDBaaS,
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	state := support.NewState(t, resource, tc.CurrentState, tc.CurrentID)
	if configured, ok := tc.CurrentState["id"]; ok {
		state.Attributes["id"] = configured.(string)
	} else {
		delete(state.Attributes, "id")
	}

	data := resource.Data(state)
	diags := resource.ReadContext(t.Context(), data, fake.TestMeta())

	return data.State(), diags
}

func projectRegion() map[string]interface{} {
	return map[string]interface{}{
		"project_id": testProjectID,
		"region_id":  testRegionID,
	}
}

func withCluster(parts map[string]interface{}) map[string]interface{} {
	out := projectRegion()
	out["cluster_id"] = testClusterID
	for key, value := range parts {
		out[key] = value
	}

	return out
}

func merge(base map[string]interface{}, parts ...map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range base {
		out[key] = value
	}

	for _, part := range parts {
		for key, value := range part {
			out[key] = value
		}
	}

	return out
}

func statusResponse(code int) *edgecloud.Response {
	return &edgecloud.Response{Response: &http.Response{StatusCode: code}}
}

func notFound() *edgecloud.Response {
	return statusResponse(http.StatusNotFound)
}

func taskResponse(ids ...string) *edgecloud.TaskResponse {
	return &edgecloud.TaskResponse{Tasks: ids}
}

func finishedTask() *edgecloud.Task {
	return &edgecloud.Task{ID: testTaskID, State: edgecloud.TaskStateFinished}
}

func failedTask() *edgecloud.Task {
	return &edgecloud.Task{ID: testTaskID, State: edgecloud.TaskStateError}
}

func ptr[T any](value T) *T {
	return &value
}
