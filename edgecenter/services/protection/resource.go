package protection

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	geoIPNo        = "no"
	geoIPAllowList = "allow"
	geoIPBlockList = "block"

	lbRoundRobin = "Round Robin"
	lbIPHash     = "Round Robin with session persistence"

	tlsv1   = "1"
	tlsv1_1 = "1.1"
	tlsv1_2 = "1.2"
	tlsv1_3 = "1.3"
)

const (
	sslCustom = "custom"
	sslLE     = "le"
)

const (
	ProtectionResourceResource = "edgecenter_protection_resource"

	ProtectionResourceSchemaActive            = "active"
	ProtectionResourceSchemaGeoIPList         = "geoip_list"
	ProtectionResourceSchemaGeoIPMode         = "geoip_mode"
	ProtectionResourceSchemaHTTPToOrigin      = "http_to_origin"
	ProtectionResourceSchemaLoadBalancingType = "load_balancing_type"
	ProtectionResourceSchemaMultipleOrigins   = "multiple_origins"
	ProtectionResourceSchemaName              = "name"
	ProtectionResourceSchemaRedirectToHTTPS   = "redirect_to_https"
	ProtectionResourceSchemaTLS               = "tls"
	ProtectionResourceSchemaWildcardAliases   = "wildcard_aliases"
	ProtectionResourceSchemaWAF               = "waf"
	ProtectionResourceSchemaWWWRedirect       = "www_redirect"

	ProtectionResourceSchemaClient    = "client"
	ProtectionResourceSchemaEnabled   = "enabled"
	ProtectionResourceSchemaIP        = "ip"
	ProtectionResourceSchemaStatus    = "status"
	ProtectionResourceSchemaWaitForLE = "wait_for_le"
)

func resourceProtectionResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceCreate,
		ReadContext:   resourceProtectionResourceRead,
		UpdateContext: resourceProtectionResourceUpdate,
		DeleteContext: resourceProtectionResourceDelete,
		Description:   "Represent DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ProtectionResourceSchemaActive: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable DDoS protection resource.",
			},
			ProtectionResourceSchemaGeoIPList: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Computed:    true,
				Description: "List of countries to apply geoip_mode policy to.",
			},
			ProtectionResourceSchemaGeoIPMode: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  fmt.Sprintf("Manage country access policy to control access to DDoS resource from the specified countries. Available values are `%s`, `%s`, `%s`.", geoIPNo, geoIPAllowList, geoIPBlockList),
				ValidateFunc: validation.StringInSlice([]string{geoIPNo, geoIPAllowList, geoIPBlockList}, false),
			},
			ProtectionResourceSchemaHTTPToOrigin: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether to use HTTP to make requests to the origin. If set to false (default), HTTPS is used.",
			},
			ProtectionResourceSchemaLoadBalancingType: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  fmt.Sprintf("Sets load balancing type. Available values are `%s`, `%s`.", lbRoundRobin, lbIPHash),
				ValidateFunc: validation.StringInSlice([]string{lbRoundRobin, lbIPHash}, false),
			},
			ProtectionResourceSchemaMultipleOrigins: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable Multiple origins feature.",
			},
			ProtectionResourceSchemaName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The DNS name of the DDoS protection resource.",
			},
			ProtectionResourceSchemaRedirectToHTTPS: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable from HTTP to HTTPS",
			},
			ProtectionResourceSchemaTLS: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				MinItems:    1,
				Required:    true,
				Description: fmt.Sprintf("The list of supported TLS versions. Available value: `%s`, `%s`, `%s`, `%s`.", tlsv1, tlsv1_1, tlsv1_2, tlsv1_3),
			},
			ProtectionResourceSchemaWildcardAliases: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable Wildcard aliases feature.",
			},
			ProtectionResourceSchemaWAF: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable WAF.",
			},
			ProtectionResourceSchemaWWWRedirect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable redirect from WWW to the primary domain option.",
			},

			ProtectionResourceSchemaClient: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Client ID.",
			},
			ProtectionResourceSchemaEnabled: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether resource is enabled.",
			},
			ProtectionResourceSchemaIP: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Resources's protected IP address.",
			},
			ProtectionResourceSchemaStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Show resource status.",
			},
			ProtectionResourceSchemaWaitForLE: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of seconds after which LE certificate can be issued.",
			},
		},
	}
}

func resourceProtectionResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DDoS Protection Resource creating")
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	var req protectionSDK.ResourceCreateRequest

	req.Name = d.Get(ProtectionResourceSchemaName).(string)

	tlsEnabled := d.Get(ProtectionResourceSchemaTLS).(*schema.Set).List()
	req.TLSEnabled = make([]string, len(tlsEnabled))
	for i, s := range tlsEnabled {
		req.TLSEnabled[i] = s.(string)
	}

	if active, ok := d.GetOk(ProtectionResourceSchemaActive); ok {
		req.Active = active.(bool)
	}

	if multipleOrigins, ok := d.GetOk(ProtectionResourceSchemaMultipleOrigins); ok {
		req.MultipleOrigins = multipleOrigins.(bool)
	}

	if wildcardAliases, ok := d.GetOk(ProtectionResourceSchemaWildcardAliases); ok {
		req.WidlcardAliases = wildcardAliases.(bool)
	}

	if redirectToHTTPS, ok := d.GetOk(ProtectionResourceSchemaRedirectToHTTPS); ok {
		req.RedirectToHTTPS = redirectToHTTPS.(bool)
	}

	if httpToOriginValue, ok := d.GetOk(ProtectionResourceSchemaHTTPToOrigin); ok {
		if httpToOriginValue.(bool) {
			req.HTTPS2HTTP = 1
		} else {
			req.HTTPS2HTTP = 0
		}
	}

	if lbType, ok := d.GetOk(ProtectionResourceSchemaLoadBalancingType); ok {
		switch lbType.(string) {
		case lbRoundRobin:
			req.IPHash = 0
		case lbIPHash:
			req.IPHash = 1
		}
	}

	if geoIPMode, ok := d.GetOk(ProtectionResourceSchemaGeoIPMode); ok {
		switch geoIPMode.(string) {
		case geoIPNo:
			req.GeoIPMode = 0
		case geoIPAllowList:
			req.GeoIPMode = 1
		case geoIPBlockList:
			req.GeoIPMode = 2
		}
	}

	if geoIPList, ok := d.GetOk(ProtectionResourceSchemaGeoIPList); ok {
		iplist := geoIPList.(*schema.Set).List()
		geoIPListSet := make([]string, len(iplist))
		for i, s := range iplist {
			geoIPListSet[i] = s.(string)
		}
		req.GeoIPList = strings.Join(geoIPListSet, ",")
	}

	if redirectValue, ok := d.GetOk(ProtectionResourceSchemaWWWRedirect); ok {
		if redirectValue.(bool) {
			req.WWWRedir = 1
		} else {
			req.WWWRedir = 0
		}
	}

	if waf, ok := d.GetOk(ProtectionResourceSchemaWAF); ok {
		req.WAF = waf.(bool)
	}

	result, _, err := client.Resources.Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	resourceProtectionResourceRead(ctx, d, m)

	log.Printf("[DEBUG] Finish DDoS Protection Resource creating (id=%d)\n", result.ID)

	return nil
}

func resourceProtectionResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start DDoS Protection Resource reading (id=%s)\n", resourceID)
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, _, err := client.Resources.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(ProtectionResourceSchemaName, result.Name)
	d.Set(ProtectionResourceSchemaTLS, result.TLSEnabled)
	d.Set(ProtectionResourceSchemaActive, result.Active)
	d.Set(ProtectionResourceSchemaMultipleOrigins, result.MultipleOrigins)
	d.Set(ProtectionResourceSchemaWildcardAliases, result.WidlcardAliases)
	d.Set(ProtectionResourceSchemaRedirectToHTTPS, result.RedirectToHTTPS)
	d.Set(ProtectionResourceSchemaGeoIPList, strings.Split(result.GeoIPList, ","))
	d.Set(ProtectionResourceSchemaWAF, result.WAF)

	if result.HTTPS2HTTP == 1 {
		d.Set(ProtectionResourceSchemaHTTPToOrigin, true)
	} else {
		d.Set(ProtectionResourceSchemaHTTPToOrigin, false)
	}

	switch result.IPHash {
	case 0:
		d.Set(ProtectionResourceSchemaLoadBalancingType, lbRoundRobin)
	case 1:
		d.Set(ProtectionResourceSchemaLoadBalancingType, lbIPHash)
	}

	switch result.GeoIPMode {
	case 0:
		d.Set(ProtectionResourceSchemaGeoIPMode, geoIPNo)
	case 1:
		d.Set(ProtectionResourceSchemaGeoIPMode, geoIPAllowList)
	case 2:
		d.Set(ProtectionResourceSchemaGeoIPMode, geoIPBlockList)
	}

	if result.WWWRedir == 1 {
		d.Set(ProtectionResourceSchemaWWWRedirect, true)
	} else {
		d.Set(ProtectionResourceSchemaWWWRedirect, false)
	}

	d.Set(ProtectionResourceSchemaClient, result.ClientID)
	d.Set(ProtectionResourceSchemaEnabled, result.Enabled)
	d.Set(ProtectionResourceSchemaIP, result.ServiceIP)
	d.Set(ProtectionResourceSchemaStatus, result.Status)
	d.Set(ProtectionResourceSchemaWaitForLE, result.WaitForLE)

	log.Println("[DEBUG] Finish DDoS Protection Resource reading")

	return nil
}

func resourceProtectionResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start DDoS Protection Resource updating (id=%s)\n", resourceID)
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, _, err := client.Resources.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.ResourceUpdateRequest

	req.SSLType = result.SSLType

	tlsEnabled := d.Get(ProtectionResourceSchemaTLS).(*schema.Set).List()
	req.TLSEnabled = make([]string, len(tlsEnabled))
	for i, s := range tlsEnabled {
		req.TLSEnabled[i] = s.(string)
	}

	if active, ok := d.GetOk(ProtectionResourceSchemaActive); ok {
		req.Active = active.(bool)
	}

	if multipleOrigins, ok := d.GetOk(ProtectionResourceSchemaMultipleOrigins); ok {
		req.MultipleOrigins = multipleOrigins.(bool)
	}

	if wildcardAliases, ok := d.GetOk(ProtectionResourceSchemaWildcardAliases); ok {
		req.WidlcardAliases = wildcardAliases.(bool)
	}

	if redirectToHTTPS, ok := d.GetOk(ProtectionResourceSchemaRedirectToHTTPS); ok {
		req.RedirectToHTTPS = redirectToHTTPS.(bool)
	}

	if httpToOriginValue, ok := d.GetOk(ProtectionResourceSchemaHTTPToOrigin); ok {
		if httpToOriginValue.(bool) {
			req.HTTPS2HTTP = 1
		} else {
			req.HTTPS2HTTP = 0
		}
	}

	if lbType, ok := d.GetOk(ProtectionResourceSchemaLoadBalancingType); ok {
		switch lbType.(string) {
		case lbRoundRobin:
			req.IPHash = 0
		case lbIPHash:
			req.IPHash = 1
		}
	}

	if geoIPMode, ok := d.GetOk(ProtectionResourceSchemaGeoIPMode); ok {
		switch geoIPMode.(string) {
		case geoIPNo:
			req.GeoIPMode = 0
		case geoIPAllowList:
			req.GeoIPMode = 1
		case geoIPBlockList:
			req.GeoIPMode = 2
		}
	}

	if geoIPList, ok := d.GetOk(ProtectionResourceSchemaGeoIPList); ok {
		iplist := geoIPList.(*schema.Set).List()
		geoIPListSet := make([]string, len(iplist))
		for i, s := range iplist {
			geoIPListSet[i] = s.(string)
		}
		req.GeoIPList = strings.Join(geoIPListSet, ",")
	}

	if redirectValue, ok := d.GetOk(ProtectionResourceSchemaWWWRedirect); ok {
		if redirectValue.(bool) {
			req.WWWRedir = 1
		} else {
			req.WWWRedir = 0
		}
	}

	if waf, ok := d.GetOk(ProtectionResourceSchemaWAF); ok {
		req.WAF = waf.(bool)
	}

	if _, _, err := client.Resources.Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish DDoS Protection Resource updating")

	return resourceProtectionResourceRead(ctx, d, m)
}

func resourceProtectionResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start DDoS Protection Resource deleting (id=%s)\n", resourceID)
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := client.Resources.Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish DDoS Protection Resource deleting")

	return nil
}
