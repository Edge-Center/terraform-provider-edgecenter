//go:build integration

package mkaas_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	mkaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/mkaas/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testProjectID    = 1
	testProjectIDStr = "1"
	testRegionID     = 8
	testRegionIDStr  = "8"

	testClusterID    = 42
	testClusterIDStr = "42"
	testClusterName  = "prod-cluster"

	testPoolID    = 7
	testPoolIDStr = "7"
	testPoolName  = "workers"

	testTaskID = "6e3a1b0e-5f2d-4c8a-9a1b-3d3d3d3d3d3d"

	testKeypairName = "ops-key"
	testNetworkID   = "0b0b0b0b-1111-2222-3333-444444444444"
	testSubnetID    = "1c1c1c1c-1111-2222-3333-444444444444"

	testFlavor        = "g1-standard-2-4"
	testPodSubnet     = "10.100.0.0/16"
	testServiceSubnet = "10.200.0.0/16"

	testShortVersion = "v1.31"
	testFullVersion  = "v1.31.4"

	unsetDataSourceID = "-"
)

func mkaasResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func mkaasDataSource(t *testing.T, name string) *schema.Resource {
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
	tc support.ResourceCase[*mkaasmock.MockedMKaaS],
	fake *mkaasmock.MockedMKaaS,
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

func finishedTask(created map[string]interface{}) *edgecloud.Task {
	return &edgecloud.Task{
		ID:               testTaskID,
		State:            edgecloud.TaskStateFinished,
		CreatedResources: created,
	}
}

func failedTask() *edgecloud.Task {
	return &edgecloud.Task{ID: testTaskID, State: edgecloud.TaskStateError}
}

func failedTaskWithError(text string) *edgecloud.Task {
	return &edgecloud.Task{ID: testTaskID, State: edgecloud.TaskStateError, Error: &text}
}

func clusterCreated(ids ...float64) map[string]interface{} {
	return map[string]interface{}{"mkaasclusters": ids}
}

func poolCreated(ids ...float64) map[string]interface{} {
	return map[string]interface{}{"mkaaspools": ids}
}

func availableVersions(versions ...string) *edgecloud.MKaaSKubernetesVersionsResult {
	items := make([]edgecloud.MKaaSKubernetesVersion, 0, len(versions))
	for _, version := range versions {
		items = append(items, edgecloud.MKaaSKubernetesVersion{Version: version})
	}

	return &edgecloud.MKaaSKubernetesVersionsResult{Versions: items}
}

func ptr[T any](value T) *T {
	return &value
}

func marshal(t *testing.T, value interface{}) string {
	t.Helper()

	body, err := json.Marshal(value)
	require.NoError(t, err)

	return string(body)
}
