//go:build websec

package edgecenter_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccWebsecSSLCertificate(t *testing.T) {
	t.Parallel()

	resourceName := "edgecenter_websec_ssl_certificate.acctest"
	name := testName("websec-cert")
	firstCert, firstKey := websecSelfSignedCertificate(t, "first.example.com")
	secondCert, secondKey := websecSelfSignedCertificate(t, "second.example.com")

	template := func(name, cert, key string) string {
		return fmt.Sprintf(`
resource "edgecenter_websec_ssl_certificate" "acctest" {
  name        = %q
  certificate = %q
  private_key = %q
}
`, name, cert, key)
	}

	var firstID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_WEBSEC_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(name, firstCert, firstKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttrWith(resourceName, "id", func(id string) error {
						firstID = id

						return nil
					}),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "automated", "false"),
					resource.TestCheckResourceAttr(resourceName, "has_related_resources", "false"),
					resource.TestCheckResourceAttr(resourceName, "cert_subject_cn", "first.example.com"),
					resource.TestCheckResourceAttrSet(resourceName, "validity_not_after"),
				),
			},
			{
				Config: template(name+"-renamed", firstCert, firstKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPtr(resourceName, "id", &firstID),
					resource.TestCheckResourceAttr(resourceName, "name", name+"-renamed"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate", "private_key", "certificates_chain"},
			},
			{
				Config: template(name+"-renamed", secondCert, secondKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith(resourceName, "id", func(id string) error {
						if id == firstID {
							return errors.New("new material must replace the certificate")
						}

						return nil
					}),
					resource.TestCheckResourceAttr(resourceName, "cert_subject_cn", "second.example.com"),
				),
			},
		},
	})
}

func websecSelfSignedCertificate(t *testing.T, commonName string) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		DNSNames:     []string{commonName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return string(cert), string(keyPEM)
}
