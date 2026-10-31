package websec

import (
	"context"
	"encoding/pem"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

var sslPrivateKeyPEMTypes = []string{"PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY", "DSA PRIVATE KEY"}

func resourceWebsecSSLCertificate() *schema.Resource {
	return &schema.Resource{
		Description: "WebSec SSL certificate: your own certificate that `edgecenter_websec_resource` uses through " +
			"`ssl_certificate`. Certificates the server issues itself, such as Let's Encrypt, are managed by " +
			"`edgecenter_websec_lecert`.\n\n" +
			"The API cannot change the certificate, the private key or the chain of an existing certificate, so changing " +
			"any of them replaces the certificate. Only `name` is updated in place. The API does not return them either, " +
			"and an imported certificate has none of them in state. The first apply after import only writes them from the " +
			"configuration to state and sends nothing to the API, so make sure they are the ones the certificate was " +
			"uploaded with: for example, compare `cert_subject_cn` and `validity_not_after` with the output of " +
			"`openssl x509 -noout -subject -enddate`.\n\n" +
			"A certificate that resources still use is not deleted: the API would delete them together with the " +
			"certificate, so the provider stops and lists them. To replace a certificate in use in one apply, keep " +
			"`create_before_destroy` in its `lifecycle` from the start, so that the resources move to the new certificate " +
			"before the old one is deleted, and derive its name from the certificate, because names are unique among the " +
			"certificates of the client. The check does not see subdomains: a certificate moved from the old protection " +
			"API can be attached to a subdomain, and deleting such a certificate deletes the subdomain.\n\n" +
			"Import with `<cert_id>` or `resource:<resource_id>`. `GET <edgecenter_websec_api>/ssl/certificates` lists the " +
			"certificates with their ids, and `edgecenter_websec_api` defaults to `<api_endpoint>/v2/protection`. " +
			"`resource:<resource_id>` imports the certificate that `edgecenter_websec_resource` uses. A resource moved from " +
			"the old protection API keeps its id, and the id of `edgecenter_protection_resource_certificate` in the old " +
			"state is that resource id, so the second form needs only the old state. A certificate the server issued is not " +
			"imported here: import `edgecenter_websec_lecert` by the resource id instead.",
		CreateContext: resourceWebsecSSLCertificateCreate,
		ReadContext:   resourceWebsecSSLCertificateRead,
		UpdateContext: resourceWebsecSSLCertificateUpdate,
		DeleteContext: resourceWebsecSSLCertificateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importWebsecSSLCertificate,
		},
		CustomizeDiff: replaceCertificateOnMaterialChange,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.All(validation.StringLenBetween(1, 255), validateTrimmed),
				Description: "Name of the certificate, up to 255 characters, unique among the certificates of the client. " +
					"Changing it renames the certificate in place.",
			},
			"certificate": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateCertificatesPEM,
				Description: "The certificate in PEM format. When it holds several certificates, each starts on a new line. " +
					"Changing it replaces the certificate.",
			},
			"private_key": {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: validatePrivateKeyPEM,
				Description: "The private key in PEM format: `PRIVATE KEY`, `RSA PRIVATE KEY`, `EC PRIVATE KEY` or " +
					"`DSA PRIVATE KEY`, not encrypted. It is stored in the state in plain text. Changing it replaces the " +
					"certificate.",
			},
			"certificates_chain": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCertificatesChainPEM,
				Description:  "Intermediate certificates in PEM format. Adding, changing or removing the chain replaces the certificate.",
			},
			"automated": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the server issued the certificate itself. It is `false` for an uploaded certificate.",
			},
			"has_related_resources": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether a resource uses the certificate. Subdomains are not taken into account.",
			},
			"cert_issuer": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Common name of the certificate issuer.",
			},
			"cert_subject_cn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Common name of the certificate subject.",
			},
			"cert_subject_alt": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Subject alternative names of the certificate.",
			},
			"validity_not_before": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Start of the certificate validity period in RFC 3339 format.",
			},
			"validity_not_after": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "End of the certificate validity period in RFC 3339 format.",
			},
		},
	}
}

func importWebsecSSLCertificate(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*edgecenter.Config).WebsecClient

	id, isResourceRef, err := parseIDOrResourceRef(d.Id())
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <cert_id> or resource:<resource_id>: %w", d.Id(), err)
	}

	certID := id
	if isResourceRef {
		resource, err := client.Resources().Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("read resource %d: %w", id, err)
		}
		if resource.SSLCertificate == nil {
			return nil, fmt.Errorf("resource %d has no SSL certificate", id)
		}
		certID = *resource.SSLCertificate
	}

	cert, err := client.SSLCerts().Get(ctx, certID)
	if err != nil {
		if isResourceRef && isNotFound(err) {
			return nil, fmt.Errorf("resource %d refers to SSL certificate %d, which has been deleted; import the certificate "+
				"by its own id instead", id, certID)
		}

		return nil, fmt.Errorf("read SSL certificate %d: %w", certID, err)
	}
	if cert.Automated {
		return nil, fmt.Errorf("SSL certificate %d is issued by the server: import edgecenter_websec_lecert by the "+
			"resource id instead", certID)
	}

	d.SetId(strconv.FormatInt(certID, 10))

	return []*schema.ResourceData{d}, nil
}

func replaceCertificateOnMaterialChange(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if oldCertificate, _ := d.GetChange("certificate"); oldCertificate.(string) == "" {
		return nil
	}

	for _, key := range []string{"certificate", "private_key", "certificates_chain"} {
		if !d.HasChange(key) {
			continue
		}
		if err := d.ForceNew(key); err != nil {
			return fmt.Errorf("mark %s change as force new: %w", key, err)
		}
	}

	return nil
}

func resourceWebsecSSLCertificateCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start WebSec SSL certificate creating")
	client := m.(*edgecenter.Config).WebsecClient

	cert, err := client.SSLCerts().Create(ctx, &sslcerts.CreateRequest{
		Name:              d.Get("name").(string),
		Certificate:       d.Get("certificate").(string),
		PrivateKey:        d.Get("private_key").(string),
		CertificatesChain: d.Get("certificates_chain").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(cert.ID, 10))
	log.Printf("[DEBUG] Finish WebSec SSL certificate creating (id=%d)\n", cert.ID)

	return resourceWebsecSSLCertificateRead(ctx, d, m)
}

func resourceWebsecSSLCertificateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec SSL certificate reading (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cert, err := client.SSLCerts().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec SSL certificate %d not found, removing it from state\n", id)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	d.Set("name", cert.Name)
	d.Set("automated", cert.Automated)
	d.Set("has_related_resources", cert.HasRelatedResources)
	d.Set("cert_issuer", cert.CertIssuer)
	d.Set("cert_subject_cn", cert.CertSubjectCN)
	d.Set("cert_subject_alt", cert.CertSubjectAlt)
	d.Set("validity_not_before", formatSSLCertificateTime(cert.ValidityNotBefore))
	d.Set("validity_not_after", formatSSLCertificateTime(cert.ValidityNotAfter))

	log.Printf("[DEBUG] Finish WebSec SSL certificate reading (id=%d)\n", id)

	return nil
}

func resourceWebsecSSLCertificateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec SSL certificate updating (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		if _, err := client.SSLCerts().Update(ctx, id, &sslcerts.UpdateRequest{Name: d.Get("name").(string)}); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Finish WebSec SSL certificate updating (id=%d)\n", id)

	return resourceWebsecSSLCertificateRead(ctx, d, m)
}

func resourceWebsecSSLCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec SSL certificate deleting (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cert, err := client.SSLCerts().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec SSL certificate %d is already deleted\n", id)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	if len(cert.RelatedResources) > 0 {
		users := make([]string, 0, len(cert.RelatedResources))
		for _, related := range cert.RelatedResources {
			users = append(users, fmt.Sprintf("resource %d (%s)", related.ID, related.Domain))
		}

		return diag.Errorf("SSL certificate %d is still used by %s: remove it from them first, or keep "+
			"create_before_destroy on it to replace it, because the API deletes a certificate together with the resources "+
			"that use it",
			id, strings.Join(users, ", "))
	}

	if err := client.SSLCerts().Delete(ctx, id); err != nil && !isNotFound(err) {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish WebSec SSL certificate deleting (id=%d)\n", id)

	return nil
}

func formatSSLCertificateTime(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.Format(time.RFC3339)
}

func pemBlockTypes(value string) []string {
	var types []string

	rest := []byte(value)
	for {
		block, next := pem.Decode(rest)
		if block == nil {
			return types
		}

		types = append(types, block.Type)
		rest = next
	}
}

func validateCertificatesPEM(i interface{}, k string) ([]string, []error) {
	types := pemBlockTypes(i.(string))
	if len(types) == 0 {
		return nil, []error{fmt.Errorf("expected %s to be PEM encoded certificates", k)}
	}

	for _, blockType := range types {
		if blockType != "CERTIFICATE" {
			return nil, []error{fmt.Errorf("expected %s to hold only CERTIFICATE PEM blocks, got %s", k, blockType)}
		}
	}

	return nil, nil
}

func validateCertificatesChainPEM(i interface{}, k string) ([]string, []error) {
	if i.(string) == "" {
		return nil, nil
	}

	return validateCertificatesPEM(i, k)
}

func validatePrivateKeyPEM(i interface{}, k string) ([]string, []error) {
	types := pemBlockTypes(i.(string))
	if len(types) == 1 && slices.Contains(sslPrivateKeyPEMTypes, types[0]) {
		return nil, nil
	}

	got := "no PEM block"
	if len(types) > 0 {
		got = strings.Join(types, ", ")
	}

	return nil, []error{fmt.Errorf("expected %s to be a single unencrypted PEM key of type PRIVATE KEY, RSA PRIVATE KEY, "+
		"EC PRIVATE KEY or DSA PRIVATE KEY, got %s", k, got)}
}
