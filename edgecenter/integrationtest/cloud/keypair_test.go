//go:build unit

package edgecenter_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
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

func TestUnitKeypair_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_keypair"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		keypairCreateCase(testKeypairID),
		keypairDeleteCase(testKeypairID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
