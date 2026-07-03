//go:build integration

package edgecenter_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const (
	testSecretID   = "secret-1"
	testSecretName = "test-cert"
)

func sampleSecret(id, name, algorithm, status string) *edgecloud.Secret {
	return &edgecloud.Secret{
		ID:         id,
		Name:       name,
		Algorithm:  algorithm,
		BitLength:  2048,
		Mode:       "CBC",
		Status:     status,
		Created:    "2026-01-15T10:00:00.180394",
		Expiration: "",
		ContentTypes: map[string]string{
			"certificate":       "application/octet-stream",
			"private_key":       "application/octet-stream",
			"certificate_chain": "application/octet-stream",
		},
	}
}

func secretCreateCase(secretID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Secrets.On("CreateV2", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SecretCreateRequestV2) bool {
			return req.Name == "test-cert" && req.Payload.Certificate == "cert-pem"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-cre-secret"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-cre-secret").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"secrets": []interface{}{secretID},
			},
		}, nil, nil)

	mc.Secrets.On("Get", mock.Anything, secretID).
		Return(sampleSecret(secretID, "test-cert", "RSA", "ACTIVE"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-cert"),
			map[string]interface{}{
				"private_key":       "priv-key-pem",
				"certificate":       "cert-pem",
				"certificate_chain": "chain-pem",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secretID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      "test-cert",
				"algorithm": "RSA",
				"status":    "ACTIVE",
			})
		},
	}
}

func secretReadCase(secretID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Secrets.On("Get", mock.Anything, secretID).
		Return(sampleSecret(secretID, "test-cert", "RSA", "ACTIVE"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing secret",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: secretID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-cert"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, secretID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":      "test-cert",
				"algorithm": "RSA",
				"status":    "ACTIVE",
			})
		},
	}
}

func secretReadNotFoundCase(secretID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Secrets.On("Get", mock.Anything, secretID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: secretID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
			require.NotNil(t, state, "state is preserved on read error")
		},
	}
}

func secretDeleteCase(secretID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Secrets.On("Delete", mock.Anything, secretID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del-secret"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del-secret").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete secret",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: secretID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func secretCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Secrets.On("CreateV2", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("api error: cannot create secret"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-secret"),
			map[string]interface{}{
				"private_key":       "pk",
				"certificate":       "cert",
				"certificate_chain": "chain",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "cannot create secret")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func secretDeleteTaskErrorCase(secretID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Secrets.On("Delete", mock.Anything, secretID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-del-secret-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-del-secret-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: secretID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, secretID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationSecret_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_secret"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		secretCreateCase(testSecretID),
		secretReadCase(testSecretID),
		secretReadNotFoundCase(testSecretID),
		secretDeleteCase(testSecretID),
		secretCreateAPIFailureCase(),
		secretDeleteTaskErrorCase(testSecretID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
