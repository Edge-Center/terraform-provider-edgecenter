package websec

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	websecsdk "github.com/Edge-Center/edgecentercdn-go/websec"
	"github.com/Edge-Center/edgecentercdn-go/websec/options"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/tfutil"
)

const (
	resourceUpdateTimeout    = 10 * time.Minute
	subdomainsLockRetryPause = 2 * time.Second
	deadlineMargin           = time.Minute
)

var (
	domainNamePattern = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+(xn--[a-z0-9]{2,}|[a-z]{2,})$`)

	validateDomainName = stringValidator(
		func(value string) bool { return len(value) <= 255 && domainNamePattern.MatchString(value) },
		"be a lowercase domain name of two levels or more and up to 255 characters, with \"*.\" allowed as the first label only",
	)
)

func resourceWebsecResource() *schema.Resource {
	return &schema.Resource{
		Description: "WebSec resource: a domain under DDoS protection with its subdomains, source group, SSL certificate and " +
			"protection options. Point the A record of the domain to `protected_ip`. For your own certificate set " +
			"`ssl_certificate`; for a Let's Encrypt one leave it unset and add `edgecenter_websec_lecert`. An update is " +
			"refused while `protected_ip` is empty, because the API would move the domain to a new address.\n\n" +
			"Changing `domain` replaces the resource; every other attribute is updated in place.\n\n" +
			"Import with `<resource_id>`. `GET <edgecenter_websec_api>/resources` lists the resources with their ids, " +
			"and `edgecenter_websec_api` defaults to `<api_endpoint>/v2/protection`. A resource moved from the old " +
			"protection API keeps the id of its `edgecenter_protection_resource`.",
		CreateContext: resourceWebsecResourceCreate,
		ReadContext:   resourceWebsecResourceRead,
		UpdateContext: resourceWebsecResourceUpdate,
		DeleteContext: resourceWebsecResourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importWebsecResource,
		},
		Timeouts: &schema.ResourceTimeout{
			Update: schema.DefaultTimeout(resourceUpdateTimeout),
		},
		CustomizeDiff: customdiff.All(
			validateSubdomainsOfDomain,
			validateOptionsNotEmpty,
			validateWWWRedirects,
			validateResponseHeaderNames,
		),
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDomainName,
				Description: "Domain name to protect, in lowercase, two levels or more. \"*.\" is allowed as the first label. " +
					"Changing it replaces the resource.",
			},
			"source_group": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "ID of the `edgecenter_websec_source_group` the requests go to.",
			},
			"subdomains": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString, ValidateFunc: validateDomainName},
				Description: "Subdomains of `domain` protected together with it, in lowercase, each ending with `.<domain>`. " +
					"\"*.\" as the first label needs wildcard subdomains enabled for the account.",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable or disable the resource. Defaults to `true`.",
			},
			"ssl_certificate": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateFunc:     validation.IntAtLeast(1),
				DiffSuppressFunc: suppressAutomatedCertificateDiff,
				Description: "ID of your own `edgecenter_websec_ssl_certificate`. Leave it unset when the certificate comes from " +
					"`edgecenter_websec_lecert`: the server links that one itself. A change is refused while an automated " +
					"certificate is linked: to switch to your own certificate, remove the `edgecenter_websec_lecert` of the " +
					"resource in the same apply.",
			},
			"options": resourceOptionsSchema(),
			"ssl_automated": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the server issues and renews the certificate of the resource.",
			},
			"ssl_certificate_automated": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the linked certificate is an automated one, issued by `edgecenter_websec_lecert`.",
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Resource status: \"processed\" after create, \"active\" once the protection agents report the " +
					"resource. A resource moved from the old protection API keeps its old status string.",
			},
			"protected_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Protected IP address. The A record of `domain` must point to it.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the resource is enabled by the administrator.",
			},
			"client": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Client ID.",
			},
		},
	}
}

func importWebsecResource(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	id, err := parseID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("import id %q must be <resource_id>: %w", d.Id(), err)
	}

	d.SetId(strconv.FormatInt(id, 10))

	return []*schema.ResourceData{d}, nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%q is not a positive integer", value)
	}

	return id, nil
}

func parseIDOrResourceRef(value string) (int64, bool, error) {
	raw, isResourceRef := strings.CutPrefix(value, "resource:")
	id, err := parseID(raw)

	return id, isResourceRef, err
}

func suppressAutomatedCertificateDiff(_, _, newValue string, d *schema.ResourceData) bool {
	if !d.Get("ssl_automated").(bool) && !d.Get("ssl_certificate_automated").(bool) {
		return false
	}

	config := d.GetRawConfig()
	if config.IsNull() {
		return newValue == "0"
	}

	return config.GetAttr("ssl_certificate").IsNull()
}

func validateSubdomainsOfDomain(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("domain") || !d.NewValueKnown("subdomains") {
		return nil
	}

	suffix := "." + d.Get("domain").(string)
	for _, subdomain := range d.Get("subdomains").(*schema.Set).List() {
		if !strings.HasSuffix(subdomain.(string), suffix) {
			return fmt.Errorf("subdomain %q does not end with %q", subdomain, suffix)
		}
	}

	return nil
}

func resourceWebsecResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start WebSec resource creating")
	client := m.(*edgecenter.Config).WebsecClient

	req := resources.CreateRequest{
		Domain:      d.Get("domain").(string),
		Subdomains:  stringsFromSet(d.Get("subdomains").(*schema.Set)),
		Active:      pointer.ToBool(d.Get("active").(bool)),
		SourceGroup: int64(d.Get("source_group").(int)),
		Options:     expandResourceOptions(d, false),
	}
	if certificate, ok := d.GetOk("ssl_certificate"); ok {
		req.SSLCertificate = pointer.ToInt64(int64(certificate.(int)))
	}

	resource, err := client.Resources().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(resource.ID, 10))
	log.Printf("[DEBUG] Finish WebSec resource creating (id=%d)\n", resource.ID)

	return resourceWebsecResourceRead(ctx, d, m)
}

func resourceWebsecResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec resource reading (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resource, err := client.Resources().Get(ctx, id)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec resource %d not found, removing it from state\n", id)
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	certificateAutomated, err := isLinkedCertificateAutomated(ctx, client, resource)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceOptions, err := flattenResourceOptions(resource.Options)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("domain", resource.Domain)
	d.Set("subdomains", resource.Subdomains)
	d.Set("active", resource.Active)
	d.Set("source_group", pointer.GetInt64(resource.SourceGroup))
	d.Set("ssl_certificate", pointer.GetInt64(resource.SSLCertificate))
	d.Set("ssl_automated", resource.SSLAutomated)
	d.Set("ssl_certificate_automated", certificateAutomated)
	d.Set("status", string(resource.Status))
	d.Set("protected_ip", pointer.GetString(resource.ProtectedIP))
	d.Set("enabled", resource.Enabled)
	d.Set("client", resource.Client)
	if err := d.Set("options", resourceOptions); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec resource reading (id=%d)\n", id)

	return nil
}

func resourceWebsecResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec resource updating (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	current, err := client.Resources().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if current.ProtectedIP == nil {
		return diag.Errorf("resource %d has no protected IP, so an update would take a new address from the pool "+
			"while the A record of %s points at the old one: ask support to restore the address first", id, current.Domain)
	}

	req := resources.UpdateRequest{
		Active:      d.Get("active").(bool),
		SourceGroup: int64(d.Get("source_group").(int)),
		Options:     expandResourceOptions(d, true),
	}

	if d.HasChange("subdomains") {
		req.Subdomains = options.Value(stringsFromSet(d.Get("subdomains").(*schema.Set)))
	}

	switch {
	case d.HasChange("ssl_certificate"):
		automated, err := isLinkedCertificateAutomated(ctx, client, current)
		if err != nil {
			return diag.FromErr(err)
		}

		if automated {
			return diag.Errorf("resource %d has an automated certificate linked, so ssl_certificate cannot be "+
				"switched: remove the edgecenter_websec_lecert of this resource in the same apply, or import it first "+
				"if Terraform does not manage it", id)
		}

		req.SSLCertificate = options.Null[int64]()
		if certificate, ok := d.GetOk("ssl_certificate"); ok {
			req.SSLCertificate = options.Value(int64(certificate.(int)))
		}
		req.SSLAutomated = pointer.ToBool(false)

		if redirect, ok := tfutil.GetOptByName(optionBlocks(d.Get("options")), "redirect_http_to_https"); ok {
			if req.Options == nil {
				req.Options = &options.ResourceOptions{}
			}
			req.Options.RedirectHTTPToHTTPS = options.Value(expandBooleanOption(redirect))
		}
	case current.SSLAutomated && current.SSLCertificate == nil:
		req.SSLAutomated = pointer.ToBool(false)
	}

	if err := updateRetryingSubdomainsLock(ctx, client.Resources(), id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish WebSec resource updating (id=%d)\n", id)

	return resourceWebsecResourceRead(ctx, d, m)
}

func resourceWebsecResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] Start WebSec resource deleting (id=%s)\n", d.Id())
	client := m.(*edgecenter.Config).WebsecClient

	id, err := parseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.Resources().Delete(ctx, id); err != nil && !isNotFound(err) {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish WebSec resource deleting (id=%d)\n", id)

	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, cdnsdk.ErrNotFound)
}

func isLinkedCertificateAutomated(ctx context.Context, client websecsdk.ClientService, resource *resources.Resource) (bool, error) {
	if resource.SSLCertificate == nil {
		return false, nil
	}

	if resource.SSLAutomated {
		return true, nil
	}

	certificate, err := client.SSLCerts().Get(ctx, *resource.SSLCertificate)
	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] WebSec resource %d links SSL certificate %d that does not exist\n", resource.ID, *resource.SSLCertificate)

			return false, nil
		}

		return false, fmt.Errorf("read SSL certificate %d of resource %d: %w", *resource.SSLCertificate, resource.ID, err)
	}

	return certificate.Automated, nil
}

func updateRetryingSubdomainsLock(ctx context.Context, service resources.Service, id int64, req *resources.UpdateRequest) error {
	retryCtx, cancel := withDeadlineMargin(ctx)
	defer cancel()

	ticker := time.NewTicker(subdomainsLockRetryPause)
	defer ticker.Stop()

	for {
		_, err := service.Update(ctx, id, req)
		if err == nil {
			return nil
		}

		err = fmt.Errorf("update resource %d: %w", id, err)
		if !isSubdomainsDeletionLocked(err) {
			return err
		}

		log.Printf("[DEBUG] Subdomains deletion is locked by another request, retrying the update of WebSec resource %d\n", id)

		if !waitTick(retryCtx, ticker) {
			return err
		}
	}
}

func withDeadlineMargin(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(ctx, deadline.Add(-deadlineMargin))
	}

	return context.WithCancel(ctx)
}

func waitTick(ctx context.Context, ticker *time.Ticker) bool {
	select {
	case <-ctx.Done():
		return false
	case <-ticker.C:
		return true
	}
}

func isSubdomainsDeletionLocked(err error) bool {
	var apiErr *cdnsdk.APIError
	if !errors.As(err, &apiErr) || !errors.Is(apiErr, cdnsdk.ErrBadRequest) {
		return false
	}

	for _, detail := range apiErr.Details {
		if detail.Field != "subdomains" {
			continue
		}

		for _, message := range detail.Messages {
			if strings.Contains(message, "temporarily blocked") {
				return true
			}
		}
	}

	return false
}
