//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/sslcerts"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testCertID   = 3001
	testCertName = "tf-cert"
	testCertBody = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"
	testCertKey  = "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----"
)

func sslCertConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":        testCertName,
		"cert":        testCertBody,
		"private_key": testCertKey,
	}
}

func sslCertCreateCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Create", mock.Anything,
		mock.MatchedBy(func(req *sslcerts.CreateRequest) bool {
			return req.Name == testCertName && req.Cert == testCertBody && req.PrivateKey == testCertKey
		}),
	).Return(&sslcerts.Cert{ID: testCertID, Name: testCertName}, nil)

	mc.SSLCerts.On("Get", mock.Anything, int64(testCertID)).
		Return(&sslcerts.Cert{ID: testCertID, Name: testCertName, HasRelatedResources: true, Automated: false}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCertID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":                  testCertName,
				"cert":                  testCertBody,
				"private_key":           testCertKey,
				"has_related_resources": "true",
				"automated":             "false",
			})
		},
	}
}

func sslCertReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Get", mock.Anything, int64(testCertID)).
		Return(&sslcerts.Cert{ID: testCertID, Name: testCertName, HasRelatedResources: false, Automated: true}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read existing cert",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCertID),
		CurrentState: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCertID))
			support.RequireStateAttrs(t, state, map[string]string{
				"has_related_resources": "false",
				"automated":             "true",
			})
		},
	}
}

func sslCertDeleteCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Delete", mock.Anything, int64(testCertID)).Return(nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "delete cert",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCertID),
		CurrentState: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func sslCertCreateAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: certificate is invalid"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *cdnmock.MockedCDN { return mc },
		NewConfig: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "certificate is invalid")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func sslCertDeleteAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Delete", mock.Anything, int64(testCertID)).
		Return(fmt.Errorf("api error: cert is in use"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on delete keeps state",
		Op:           support.OpDelete,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCertID),
		CurrentState: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "cert is in use")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCertID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func sslCertReadAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.SSLCerts.On("Get", mock.Anything, int64(testCertID)).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    fmt.Sprintf("%d", testCertID),
		CurrentState: sslCertConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must be kept when read fails")
		},
	}
}

func TestIntegrationSSLCert_TableDriven(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_sslcert")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		sslCertCreateCase(),
		sslCertReadCase(),
		sslCertDeleteCase(),
		sslCertCreateAPIFailureCase(),
		sslCertDeleteAPIFailureCase(),
		sslCertReadAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
