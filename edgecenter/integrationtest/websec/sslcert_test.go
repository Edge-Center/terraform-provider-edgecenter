//go:build integration

package websec_test

import (
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	websecmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/websec/mock"
)

const (
	sslCertName    = "example.com-3f2a9c1d"
	sslCertRenamed = "example.com-7be04d11"
)

var (
	sslCertPEM      = sslCertPEMBlock("CERTIFICATE", "leaf")
	sslCertKeyPEM   = sslCertPEMBlock("PRIVATE KEY", "key")
	sslCertChainPEM = sslCertPEMBlock("CERTIFICATE", "intermediate")
)

func sslCertPEMBlock(blockType, body string) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: []byte(body)}))
}

func sslCertConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":               sslCertName,
		"certificate":        sslCertPEM,
		"private_key":        sslCertKeyPEM,
		"certificates_chain": sslCertChainPEM,
	}
}

func sslCertConfigWith(key string, value interface{}) map[string]interface{} {
	config := sslCertConfig()
	config[key] = value

	return config
}

func sslCertConfigWithout(key string) map[string]interface{} {
	config := sslCertConfig()
	delete(config, key)

	return config
}

func sslCertFromAPI() *sslcerts.Cert {
	notBefore := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(2027, 1, 1, 12, 30, 0, 123456000, time.UTC)

	return &sslcerts.Cert{
		ID:                testCertID,
		Name:              sslCertName,
		CertIssuer:        "Example CA",
		CertSubjectCN:     "example.com",
		CertSubjectAlt:    "example.com,www.example.com",
		ValidityNotBefore: &notBefore,
		ValidityNotAfter:  &notAfter,
	}
}

func TestIntegrationWebsecSSLCertificate_Create(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	expired := apiError(t, http.StatusBadRequest, cdnsdk.ErrBadRequest,
		`{"errors": {"certificate": ["Your certificate has expired. Please install a new one."]}}`)
	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "create uploads the material and reads back what the api derives from it",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Create", mock.Anything, &sslcerts.CreateRequest{
					Name:              sslCertName,
					Certificate:       sslCertPEM,
					PrivateKey:        sslCertKeyPEM,
					CertificatesChain: sslCertChainPEM,
				}).Return(&sslcerts.Cert{ID: testCertID, Name: sslCertName}, nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()

				return mc
			},
			NewConfig: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateID(t, state, testCertIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                  sslCertName,
					"certificate":           sslCertPEM,
					"private_key":           sslCertKeyPEM,
					"certificates_chain":    sslCertChainPEM,
					"automated":             "false",
					"has_related_resources": "false",
					"cert_issuer":           "Example CA",
					"cert_subject_cn":       "example.com",
					"cert_subject_alt":      "example.com,www.example.com",
					"validity_not_before":   "2026-10-01T00:00:00Z",
					"validity_not_after":    "2027-01-01T12:30:00Z",
				})
			},
		},
		{
			Name: "a rejected create shows the api error and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Create", mock.Anything, mock.Anything).Return(nil, expired).Once()

				return mc
			},
			NewConfig: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Your certificate has expired")
				require.Nil(t, state)
			},
		},
		{
			Name: "create keeps the id when the read after it fails",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Create", mock.Anything, mock.Anything).
					Return(&sslcerts.Cert{ID: testCertID, Name: sslCertName}, nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, unavailable).Once()

				return mc
			},
			NewConfig: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testCertIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSSLCertificate_Read(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "read refreshes the name and keeps the material the api does not return",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				cert := sslCertFromAPI()
				cert.Name = sslCertRenamed
				cert.HasRelatedResources = true
				cert.RelatedResources = []sslcerts.RelatedResource{{ID: testResourceID, Domain: "example.com"}}

				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(cert, nil).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                  sslCertRenamed,
					"certificate":           sslCertPEM,
					"private_key":           sslCertKeyPEM,
					"certificates_chain":    sslCertChainPEM,
					"has_related_resources": "true",
				})
			},
		},
		{
			Name: "read removes a certificate the api cannot find from state",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "read keeps the state when the api fails for another reason",
			Op:   support.OpRead,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, unavailable).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testCertIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSSLCertificate_Update(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	renamed := sslCertConfig()
	renamed["name"] = sslCertRenamed

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "renaming sends only the new name",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				cert := sslCertFromAPI()
				cert.Name = sslCertRenamed

				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Update", mock.Anything, testCertID, &sslcerts.UpdateRequest{Name: sslCertRenamed}).
					Return(cert, nil).Once()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(cert, nil).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			NewConfig:    renamed,
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateID(t, state, testCertIDStr)
				support.RequireStateAttrs(t, state, map[string]string{"name": sslCertRenamed})
			},
		},
		{
			Name: "the first apply after import writes the material to the state and sends nothing",
			Op:   support.OpApply,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: map[string]interface{}{"name": sslCertName},
			NewConfig:    sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				support.RequireStateID(t, state, testCertIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"certificate":        sslCertPEM,
					"private_key":        sslCertKeyPEM,
					"certificates_chain": sslCertChainPEM,
				})
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSSLCertificate_Delete(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	unavailable := serverError(t)

	cases := []support.ResourceCase[*websecmock.MockedWebsec]{
		{
			Name: "delete removes a certificate no resource uses",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()
				mc.SSLCerts.On("Delete", mock.Anything, testCertID).Return(nil).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete refuses while resources use the certificate and names them",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				cert := sslCertFromAPI()
				cert.HasRelatedResources = true
				cert.RelatedResources = []sslcerts.RelatedResource{
					{ID: testResourceID, Domain: "example.com"},
					{ID: 43, Domain: "shop.example.com"},
				}

				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(cert, nil).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags,
					"SSL certificate 11 is still used by resource 42 (example.com), resource 43 (shop.example.com):")
				support.RequireErrorDiagContains(t, diags, "create_before_destroy")
				support.RequireStateID(t, state, testCertIDStr)
			},
		},
		{
			Name: "delete of a certificate that is already gone succeeds without a delete call",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, notFound()).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete treats a 404 from the delete call as success",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()
				mc.SSLCerts.On("Delete", mock.Anything, testCertID).Return(notFound()).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "delete keeps the state when the api fails",
			Op:   support.OpDelete,
			Prepare: func() *websecmock.MockedWebsec {
				mc := websecmock.NewMockedWebsec()
				mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()
				mc.SSLCerts.On("Delete", mock.Anything, testCertID).Return(unavailable).Once()

				return mc
			},
			CurrentID:    testCertIDStr,
			CurrentState: sslCertConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *websecmock.MockedWebsec) {
				support.RequireErrorDiagContains(t, diags, "Error occurred.")
				support.RequireStateID(t, state, testCertIDStr)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase)
}

func TestIntegrationWebsecSSLCertificate_Validate(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	t.Run("the key formats the api reads, a certificate followed by its chain and an empty chain pass validation", func(t *testing.T) {
		t.Parallel()

		for i, config := range []map[string]interface{}{
			sslCertConfigWith("private_key", sslCertPEMBlock("PRIVATE KEY", "key")),
			sslCertConfigWith("private_key", sslCertPEMBlock("RSA PRIVATE KEY", "key")),
			sslCertConfigWith("private_key", sslCertPEMBlock("EC PRIVATE KEY", "key")),
			sslCertConfigWith("private_key", sslCertPEMBlock("DSA PRIVATE KEY", "key")),
			sslCertConfigWith("certificate", sslCertPEM+sslCertChainPEM),
			sslCertConfigWith("certificates_chain", ""),
		} {
			require.Emptyf(t, resource.Validate(terraform.NewResourceConfigRaw(config)), "config %d", i)
		}
	})

	rejected := []struct {
		name    string
		config  map[string]interface{}
		message string
	}{
		{
			name:    "a name with trailing whitespace",
			config:  sslCertConfigWith("name", sslCertName+" "),
			message: `expected name to have no leading or trailing whitespace (the API trims it), got "example.com-3f2a9c1d "`,
		},
		{
			name:    "a key in place of the certificate",
			config:  sslCertConfigWith("certificate", sslCertKeyPEM),
			message: "expected certificate to hold only CERTIFICATE PEM blocks, got PRIVATE KEY",
		},
		{
			name:    "a certificate without the pem armor",
			config:  sslCertConfigWith("certificate", "bGVhZg=="),
			message: "expected certificate to be PEM encoded certificates",
		},
		{
			name:    "a key the chain carries along",
			config:  sslCertConfigWith("certificates_chain", sslCertChainPEM+sslCertKeyPEM),
			message: "expected certificates_chain to hold only CERTIFICATE PEM blocks, got PRIVATE KEY",
		},
		{
			name:    "an ec key with its parameters in front",
			config:  sslCertConfigWith("private_key", sslCertPEMBlock("EC PARAMETERS", "prime256v1")+sslCertPEMBlock("EC PRIVATE KEY", "key")),
			message: "expected private_key to be a single unencrypted PEM key of type PRIVATE KEY, RSA PRIVATE KEY, EC PRIVATE KEY or DSA PRIVATE KEY, got EC PARAMETERS, EC PRIVATE KEY",
		},
		{
			name:    "an encrypted key",
			config:  sslCertConfigWith("private_key", sslCertPEMBlock("ENCRYPTED PRIVATE KEY", "key")),
			message: "got ENCRYPTED PRIVATE KEY",
		},
		{
			name:    "a key without the pem armor",
			config:  sslCertConfigWith("private_key", "a2V5"),
			message: "got no PEM block",
		},
	}

	for _, tc := range rejected {
		t.Run(tc.name+" is rejected in plan", func(t *testing.T) {
			t.Parallel()

			diags := resource.Validate(terraform.NewResourceConfigRaw(tc.config))
			support.RequireErrorDiagContains(t, diags, tc.message)
			for _, d := range diags {
				require.NotContains(t, d.Summary+d.Detail, "-----BEGIN", "a validation error must not echo the material")
			}
		})
	}
}

func TestIntegrationWebsecSSLCertificate_Import(t *testing.T) {
	t.Parallel()

	resource := websecResource(t, "edgecenter_websec_ssl_certificate")

	t.Run("the certificate id imports the certificate", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()

		imported, err := runImport(t, resource, mc, testCertIDStr)
		require.NoError(t, err)
		require.Equal(t, testCertIDStr, imported.Id())
	})

	t.Run("a resource reference imports the certificate of the resource", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).
			Return(&resources.Resource{ID: testResourceID, SSLCertificate: ptr(testCertID)}, nil).Once()
		mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(sslCertFromAPI(), nil).Once()

		imported, err := runImport(t, resource, mc, "resource:"+testResourceIDStr)
		require.NoError(t, err)
		require.Equal(t, testCertIDStr, imported.Id())
	})

	t.Run("a resource without a certificate has nothing to import", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).Return(&resources.Resource{ID: testResourceID}, nil).Once()

		_, err := runImport(t, resource, mc, "resource:"+testResourceIDStr)
		require.EqualError(t, err, "resource 42 has no SSL certificate")
	})

	t.Run("a resource that refers to a deleted certificate is explained", func(t *testing.T) {
		t.Parallel()

		mc := websecmock.NewMockedWebsec()
		mc.Resources.On("Get", mock.Anything, testResourceID).
			Return(&resources.Resource{ID: testResourceID, SSLCertificate: ptr(testCertID)}, nil).Once()
		mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(nil, notFound()).Once()

		_, err := runImport(t, resource, mc, "resource:"+testResourceIDStr)
		require.ErrorContains(t, err, "resource 42 refers to SSL certificate 11, which has been deleted")
	})

	t.Run("a certificate issued by the server is left to the lecert resource", func(t *testing.T) {
		t.Parallel()

		cert := sslCertFromAPI()
		cert.Automated = true

		mc := websecmock.NewMockedWebsec()
		mc.SSLCerts.On("Get", mock.Anything, testCertID).Return(cert, nil).Once()

		_, err := runImport(t, resource, mc, testCertIDStr)
		require.ErrorContains(t, err, "import edgecenter_websec_lecert by the resource id instead")
	})

	t.Run("a malformed id is rejected with the expected format", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "abc", "0", "resource:", "resource:-1", "resources:42"} {
			_, err := runImport(t, resource, websecmock.NewMockedWebsec(), id)
			require.ErrorContainsf(t, err, "must be <cert_id> or resource:<resource_id>", "import id %q", id)
		}
	})
}
