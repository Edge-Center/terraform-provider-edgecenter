package websec

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/websec/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	leCertPollInterval = 10 * time.Second

	leCertStatusDone   = "DONE"
	leCertStatusFailed = "FAILED"
)

func ResourceWebsecLECert(pollInterval time.Duration) *schema.Resource {
	return &schema.Resource{
		Description: "WebSec automated certificate: a certificate that the server issues and renews for an " +
			"`edgecenter_websec_resource`, from Let's Encrypt or another ACME authority (see `cert_type`). A resource has at " +
			"most one such certificate, so the id is the resource id.\n\n" +
			"Create waits until the server applies the resource and its `status` is no longer `processed`, then requests " +
			"the certificate and waits until the request ends. A resource with `active` or `enabled` off is not waited for: " +
			"the server refuses the request at once. The authority checks the domain of the resource over HTTP, so it has " +
			"to resolve to its `protected_ip`: apply this resource after DNS points there. The certificate covers the domain " +
			"only, not the subdomains of the resource. The server repeats a failed check after a pause and Create keeps " +
			"waiting: raise `timeouts.create` if the domain needs time to resolve. A resource that uses its own " +
			"`ssl_certificate` is refused: remove the attribute from the resource in the same apply.\n\n" +
			"A request that the server ends without a certificate is reported with its reason, and the certificate is " +
			"marked tainted, so the next apply cleans up and requests it again. When the create timeout runs out first, the " +
			"error shows the reason of the last failed check and the request keeps running; the next apply also replaces a " +
			"certificate issued after the timeout, `terraform untaint` keeps it.\n\n" +
			"The server renews the certificate only while `active` and `enabled` of the resource are true and its `status` " +
			"is `active`.\n\n" +
			"Destroy cancels a running request and revokes the certificate. Turn off the `redirect_http_to_https` option of " +
			"the resource first: the revoke does not check it, and the site would redirect to HTTPS without a certificate.\n\n" +
			"Changing `resource_id` or `cert_type` replaces the certificate.\n\n" +
			"Import with `<resource_id>`. A resource moved from the old protection API keeps its id, so a Let's Encrypt " +
			"certificate of `edgecenter_protection_resource_certificate` is imported here by the id from the old state.",
		CreateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceWebsecLECertCreate(ctx, d, m, pollInterval)
		},
		ReadContext: resourceWebsecLECertRead,
		DeleteContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			return resourceWebsecLECertDelete(ctx, d, m, pollInterval)
		},
		Importer: &schema.ResourceImporter{
			StateContext: importWebsecLECert,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "ID of the `edgecenter_websec_resource` to issue the certificate for. Changing it replaces the certificate.",
			},
			"cert_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  string(lecerts.CertTypeLE),
				ValidateFunc: validation.StringInSlice([]string{
					string(lecerts.CertTypeLE),
					string(lecerts.CertTypeMDDC),
					string(lecerts.CertTypeEC),
				}, false),
				DiffSuppressFunc: func(_, oldValue, newValue string, d *schema.ResourceData) bool {
					return d.Id() != "" && oldValue == "" && newValue == string(lecerts.CertTypeLE)
				},
				Description: "ACME authority that issues the certificate: `LE` (Let's Encrypt), `MDDC` or `EC`. `MDDC` and " +
					"`EC` have to be enabled for the client, and `MDDC` is issued only for a resource without subdomains. " +
					"Changing it replaces the certificate: the old one is revoked before the new request starts, so the site " +
					"has no HTTPS until the new certificate is issued, and no certificate at all if the request fails. " +
					"`create_before_destroy` does not help because a resource cannot get a second automated certificate. " +
					"The type is read from the status of the last issue request. When the API keeps no request for the " +
					"resource, as for a certificate moved from the old protection API, an imported certificate counts as " +
					"`LE`. Defaults to `LE`.",
			},
			"ssl_certificate": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "ID of the automated certificate attached to the resource, 0 until the first one is issued.",
			},
			"issue_status": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Latest status of the last issue request: `PROCESSING`, `DONE`, `FAILED`, `CANCELLED` or " +
					"`FORBIDDEN`. Empty when the request has no status yet or the API keeps no request for the resource.",
			},
			"issue_error": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Error and details of that status, empty when the status carries none.",
			},
		},
	}
}

func importWebsecLECert(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	resourceID, err := parseID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <resource_id>: %w", d.Id(), err)
	}

	d.SetId(strconv.FormatInt(resourceID, 10))
	if err := d.Set("resource_id", resourceID); err != nil {
		return nil, fmt.Errorf("set resource_id: %w", err)
	}

	return []*schema.ResourceData{d}, nil
}

func resourceWebsecLECertCreate(ctx context.Context, d *schema.ResourceData, m interface{}, pollInterval time.Duration) diag.Diagnostics {
	log.Println("[DEBUG] Start WebSec lecert creating")
	client := m.(*edgecenter.Config).WebsecClient
	resourceID := int64(d.Get("resource_id").(int))

	resource, err := client.Resources().Get(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("read resource %d: %w", resourceID, err))
	}

	if resource.SSLCertificate != nil {
		automated, err := isLinkedCertificateAutomated(ctx, client, resource)
		if err != nil {
			return diag.FromErr(err)
		}
		if automated {
			return diag.Errorf("resource %d already has the automated certificate %d: import it with "+
				"terraform import edgecenter_websec_lecert.<name> %d", resourceID, *resource.SSLCertificate, resourceID)
		}

		return diag.Errorf("resource %d uses its own certificate %d, which an issued certificate would replace: "+
			"remove ssl_certificate from edgecenter_websec_resource in the same apply", resourceID, *resource.SSLCertificate)
	}

	if resource.Active && resource.Enabled && resource.Status == resources.StatusProcessed {
		if err := waitWhileResourceProcessed(ctx, client.Resources(), resourceID, pollInterval); err != nil {
			return diag.FromErr(err)
		}
	}

	var lastRequestID int64
	lastRequest, err := client.LECerts().Get(ctx, resourceID)
	switch {
	case err == nil:
		lastRequestID = lastRequest.ID
	case !isNotFound(err):
		return diag.FromErr(fmt.Errorf("read the last certificate issue request of resource %d: %w", resourceID, err))
	}

	issue := &lecerts.IssueRequest{CertType: lecerts.CertType(d.Get("cert_type").(string))}
	if err := client.LECerts().Issue(ctx, resourceID, issue); err != nil {
		return diag.FromErr(fmt.Errorf("request a certificate for resource %d: %w", resourceID, err))
	}

	d.SetId(strconv.FormatInt(resourceID, 10))

	request, err := waitForLECertRequest(ctx, client.LECerts(), resourceID, lastRequestID, pollInterval)
	if err != nil {
		return diag.FromErr(err)
	}

	if latest := latestLECertStatus(request); latest.Status != leCertStatusDone {
		return diag.FromErr(leCertRequestFailure(resourceID, latest))
	}

	log.Printf("[DEBUG] Finish WebSec lecert creating (id=%d)\n", resourceID)

	return resourceWebsecLECertRead(ctx, d, m)
}

func resourceWebsecLECertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec lecert reading (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	resourceID, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resource, err := client.Resources().Get(ctx, resourceID)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec resource %d not found, removing its lecert from state\n", resourceID)
			d.SetId("")

			return nil
		}

		return diag.FromErr(fmt.Errorf("read resource %d: %w", resourceID, err))
	}

	var (
		certificateID int64
		running       bool
		latest        lecerts.StatusDetail
	)

	request, err := client.LECerts().Get(ctx, resourceID)
	switch {
	case err == nil && request.ID != 0:
		running = request.Active
		latest = latestLECertStatus(request)
		if request.SSLID != nil {
			certificateID = *request.SSLID
		}
		if request.CertType != nil {
			d.Set("cert_type", string(*request.CertType))
		}
	case err == nil || isNotFound(err):
		automated, err := isLinkedCertificateAutomated(ctx, client, resource)
		if err != nil {
			return diag.FromErr(err)
		}
		if automated {
			certificateID = *resource.SSLCertificate
		}
	default:
		return diag.FromErr(fmt.Errorf("read the certificate issue request of resource %d: %w", resourceID, err))
	}

	if certificateID == 0 && !running {
		log.Printf("[WARN] WebSec resource %d has no automated certificate and no running issue request, removing its lecert from state\n", resourceID)
		d.SetId("")

		return nil
	}

	d.Set("ssl_certificate", certificateID)
	d.Set("issue_status", latest.Status)
	d.Set("issue_error", leCertStatusError(latest))

	log.Printf("[DEBUG] Finish WebSec lecert reading (id=%d)\n", resourceID)

	return nil
}

func resourceWebsecLECertDelete(ctx context.Context, d *schema.ResourceData, m interface{}, pollInterval time.Duration) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec lecert deleting (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	resourceID, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := cancelLECertRequest(ctx, client.LECerts(), resourceID, pollInterval); err != nil {
		return diag.FromErr(err)
	}

	resource, err := client.Resources().Get(ctx, resourceID)
	if err != nil {
		if isNotFound(err) {
			d.SetId("")

			return nil
		}

		return diag.FromErr(fmt.Errorf("read resource %d: %w", resourceID, err))
	}

	automated, err := isLinkedCertificateAutomated(ctx, client, resource)
	if err != nil {
		return diag.FromErr(err)
	}

	if automated {
		if err := client.LECerts().Revoke(ctx, resourceID, false); err != nil && !isNotFound(err) {
			return diag.FromErr(fmt.Errorf("revoke the automated certificate of resource %d: %w", resourceID, err))
		}
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish WebSec lecert deleting (id=%d)\n", resourceID)

	return nil
}

func waitForLECertRequest(ctx context.Context, certs lecerts.Service, resourceID, lastRequestID int64, pollInterval time.Duration) (*lecerts.Status, error) {
	pollCtx, cancel := withDeadlineMargin(ctx)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var running *lecerts.Status
	for {
		request, err := certs.Get(ctx, resourceID)
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("read the certificate issue request of resource %d: %w", resourceID, err)
		}
		if err == nil && request.ID > lastRequestID {
			if !request.Active {
				return request, nil
			}
			running = request
		}

		log.Printf("[DEBUG] Waiting for the certificate issue request of WebSec resource %d\n", resourceID)

		if !waitTick(pollCtx, ticker) {
			if running == nil {
				return nil, fmt.Errorf("certificate issue request of resource %d was accepted but has not started within the create timeout", resourceID)
			}
			if attempt := latestLECertStatus(running); attempt.Status == leCertStatusFailed {
				return nil, fmt.Errorf("certificate issue request of resource %d did not end within the create timeout, "+
					"and its last attempt failed: %s", resourceID, leCertStatusError(attempt))
			}

			return nil, fmt.Errorf("certificate issue request of resource %d did not end within the create timeout", resourceID)
		}
	}
}

func waitWhileResourceProcessed(ctx context.Context, sites resources.Service, resourceID int64, pollInterval time.Duration) error {
	pollCtx, cancel := withDeadlineMargin(ctx)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		log.Printf("[DEBUG] Waiting for WebSec resource %d to leave the %s status\n", resourceID, resources.StatusProcessed)

		if !waitTick(pollCtx, ticker) {
			return fmt.Errorf("resource %d is still %s at the create timeout: the server has not applied it yet, so "+
				"the certificate was not requested", resourceID, resources.StatusProcessed)
		}

		resource, err := sites.Get(ctx, resourceID)
		if err != nil {
			return fmt.Errorf("read resource %d: %w", resourceID, err)
		}
		if resource.Status != resources.StatusProcessed {
			return nil
		}
	}
}

func cancelLECertRequest(ctx context.Context, certs lecerts.Service, resourceID int64, pollInterval time.Duration) error {
	retryCtx, cancel := withDeadlineMargin(ctx)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		request, err := certs.Get(ctx, resourceID)
		if isNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read the certificate issue request of resource %d: %w", resourceID, err)
		}
		if !request.Active {
			return nil
		}

		err = certs.Cancel(ctx, resourceID)
		if err == nil || isNotFound(err) {
			return nil
		}

		err = fmt.Errorf("cancel the certificate issue request of resource %d: %w", resourceID, err)
		if !isLECertRequestLocked(err) {
			return err
		}

		log.Printf("[DEBUG] An issue attempt holds the certificate request of WebSec resource %d, retrying the cancel\n", resourceID)

		if !waitTick(retryCtx, ticker) {
			return err
		}
	}
}

func isLECertRequestLocked(err error) bool {
	var apiErr *cdnsdk.APIError
	if !errors.As(err, &apiErr) || !errors.Is(apiErr, cdnsdk.ErrBadRequest) {
		return false
	}

	for _, detail := range apiErr.Details {
		for _, message := range detail.Messages {
			if detail.Field == "" && strings.Contains(message, `"le_issue"`) {
				return true
			}
		}
	}

	return false
}

func latestLECertStatus(request *lecerts.Status) lecerts.StatusDetail {
	var latest lecerts.StatusDetail
	for _, status := range request.Statuses {
		if status.ID > latest.ID {
			latest = status
		}
	}

	return latest
}

func leCertStatusError(status lecerts.StatusDetail) string {
	switch {
	case status.Error == "":
		return status.Details
	case status.Details == "":
		return status.Error
	default:
		return status.Error + ": " + status.Details
	}
}

func leCertRequestFailure(resourceID int64, status lecerts.StatusDetail) error {
	if reason := leCertStatusError(status); reason != "" {
		return fmt.Errorf("certificate issue request of resource %d ended with %s: %s", resourceID, status.Status, reason)
	}

	return fmt.Errorf("certificate issue request of resource %d ended with %s", resourceID, status.Status)
}
