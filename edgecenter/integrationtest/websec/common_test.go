//go:build integration

package websec_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

const (
	testResourceID    int64 = 42
	testResourceIDStr       = "42"

	testSourceGroupID    int64 = 7
	testSourceGroupIDStr       = "7"

	testCertID    int64 = 11
	testCertIDStr       = "11"
)

func websecResource(t *testing.T, name string) *schema.Resource {
	t.Helper()

	res := provider.Provider().ResourcesMap[name]
	if res == nil {
		t.Fatalf("resource %q is not registered in the provider", name)
	}

	return res
}

func notFound() error {
	return fmt.Errorf("request: %w", cdnsdk.NewAPIError(http.StatusNotFound, cdnsdk.ErrNotFound))
}

func apiError(t *testing.T, status int, sentinel error, body string) error {
	t.Helper()

	apiErr := cdnsdk.NewAPIError(status, sentinel)
	require.NoError(t, json.Unmarshal([]byte(body), apiErr))

	return fmt.Errorf("request: %w", apiErr)
}

func serverError(t *testing.T) error {
	t.Helper()

	return apiError(t, http.StatusServiceUnavailable, nil, `{"error": "Error occurred."}`)
}

func requestJSON(t *testing.T, request interface{}) string {
	t.Helper()

	raw, err := json.Marshal(request)
	require.NoError(t, err)

	return string(raw)
}

func runImport(t *testing.T, resource *schema.Resource, mc *websecmock.MockedWebsec, id string) (*schema.ResourceData, error) {
	t.Helper()
	t.Cleanup(func() { mc.MockCleanup(t) })

	data := resource.Data(nil)
	data.SetId(id)

	imported, err := resource.Importer.StateContext(t.Context(), data, mc.TestMeta())
	if err != nil {
		return nil, err
	}
	require.Len(t, imported, 1)

	return imported[0], nil
}

func optionsConfig(blocks map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"options": []interface{}{blocks}}
}

func option(fields map[string]interface{}) []interface{} {
	return []interface{}{fields}
}

func policyOption(policyType string, values ...interface{}) []interface{} {
	return option(map[string]interface{}{"policy_type": policyType, "excepted_values": values})
}

func ptr[T any](value T) *T {
	return &value
}
