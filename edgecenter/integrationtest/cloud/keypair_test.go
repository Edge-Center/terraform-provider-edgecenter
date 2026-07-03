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
	testKeypairID   = "kp-id"
	testFingerprint = "sha256:testfingerprint123"
	testPublicKey   = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC4TestPublicKey"
)

func sampleKeypair(id, name, publicKey, fingerprint string) *edgecloud.KeyPairV2 {
	return &edgecloud.KeyPairV2{
		SSHKeyID:    id,
		SSHKeyName:  name,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
	}
}

func keypairCreateCase(kpID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.KeyPairs.On("CreateV2", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.KeyPairCreateRequestV2) bool {
			return req.SSHKeyName == "test-key" && req.PublicKey == testPublicKey
		}),
	).Return(sampleKeypair(kpID, "test-key", testPublicKey, testFingerprint), nil, nil)

	mc.KeyPairs.On("GetV2", mock.Anything, kpID).
		Return(sampleKeypair(kpID, "test-key", testPublicKey, testFingerprint), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, kpID)
			support.RequireStateAttrs(t, state, map[string]string{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
				"fingerprint": testFingerprint,
				"sshkey_id":   kpID,
			})
		},
	}
}

func keypairDeleteCase(kpID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.KeyPairs.On("DeleteV2", mock.Anything, kpID).
		Return(nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete keypair",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: kpID,
		CurrentState: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func keypairReadCase(kpID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.KeyPairs.On("GetV2", mock.Anything, kpID).
		Return(sampleKeypair(kpID, "test-key", testPublicKey, testFingerprint), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing keypair",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: kpID,
		CurrentState: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, kpID)
			support.RequireStateAttrs(t, state, map[string]string{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
				"fingerprint": testFingerprint,
				"sshkey_id":   kpID,
			})
		},
	}
}

func keypairCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.KeyPairs.On("CreateV2", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.KeyPairCreateRequestV2) bool {
			return req.SSHKeyName == "fail-key"
		}),
	).Return(nil, nil, fmt.Errorf("api error: limit exceeded"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "fail-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "limit exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func keypairDeleteAPIFailureCase(kpID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.KeyPairs.On("DeleteV2", mock.Anything, kpID).
		Return(nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "API error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: kpID,
		CurrentState: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, kpID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func keypairReadNotFoundCase(kpID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.KeyPairs.On("GetV2", mock.Anything, kpID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: kpID,
		CurrentState: cloud.Merge(
			cloud.WithProjectID(testProjectID),
			map[string]interface{}{
				"sshkey_name": "test-key",
				"public_key":  testPublicKey,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
		},
	}
}

func TestIntegrationKeypair_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_keypair"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		keypairCreateCase(testKeypairID),
		keypairReadCase(testKeypairID),
		keypairCreateAPIFailureCase(),
		keypairDeleteCase(testKeypairID),
		keypairDeleteAPIFailureCase(testKeypairID),
		keypairReadNotFoundCase(testKeypairID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
