package edgecenter

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
	// These constants are used in certificate and alias resource.
	sslCustom = "custom"
	sslLE     = "le"
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
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable DDoS protection resource.",
			},
			"geoip_list": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Computed:    true,
				Description: "List of countries to apply geoip_mode policy to.",
			},
			"geoip_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  fmt.Sprintf("Manage country access policy to control access to DDoS resource from the specified countries. Available values are `%s`, `%s`, `%s`.", geoIPNo, geoIPAllowList, geoIPBlockList),
				ValidateFunc: validation.StringInSlice([]string{geoIPNo, geoIPAllowList, geoIPBlockList}, false),
			},
			"http_to_origin": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether to use HTTP to make requests to the origin. If set to false (default), HTTPS is used.",
			},
			"load_balancing_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  fmt.Sprintf("Sets load balancing type. Available values are `%s`, `%s`.", lbRoundRobin, lbIPHash),
				ValidateFunc: validation.StringInSlice([]string{lbRoundRobin, lbIPHash}, false),
			},
			"multiple_origins": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable Multiple origins feature.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The DNS name of the DDoS protection resource.",
			},
			"redirect_to_https": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable from HTTP to HTTPS",
			},
			"tls": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				MinItems:    1,
				Required:    true,
				Description: fmt.Sprintf("The list of supported TLS versions. Available value: `%s`, `%s`, `%s`, `%s`.", tlsv1, tlsv1_1, tlsv1_2, tlsv1_3),
			},
			"wildcard_aliases": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable Wildcard aliases feature.",
			},
			"waf": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable WAF.",
			},
			"www_redirect": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable redirect from WWW to the primary domain option.",
			},

			// computed
			"client": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Client ID.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether resource is enabled.",
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Resources's protected IP address.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Show resource status.",
			},
			"wait_for_le": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of seconds after which LE certificate can be issued.",
			},
		},
	}
}

func resourceProtectionResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DDoS Protection Resource creating")
	config := m.(*Config)
	client := config.ProtectionClient

	var req protectionSDK.ResourceCreateRequest

	req.Name = d.Get("name").(string)

	tlsEnabled := d.Get("tls").(*schema.Set).List()
	req.TLSEnabled = make([]string, len(tlsEnabled))
	for i, s := range tlsEnabled {
		req.TLSEnabled[i] = s.(string)
	}

	if active, ok := d.GetOk("active"); ok {
		req.Active = active.(bool)
	}

	if multipleOrigins, ok := d.GetOk("multiple_origins"); ok {
		req.MultipleOrigins = multipleOrigins.(bool)
	}

	if wildcardAliases, ok := d.GetOk("wildcard_aliases"); ok {
		req.WidlcardAliases = wildcardAliases.(bool)
	}

	if redirectToHTTPS, ok := d.GetOk("redirect_to_https"); ok {
		req.RedirectToHTTPS = redirectToHTTPS.(bool)
	}

	if httpToOriginValue, ok := d.GetOk("http_to_origin"); ok {
		if httpToOriginValue.(bool) {
			req.HTTPS2HTTP = 1
		} else {
			req.HTTPS2HTTP = 0
		}
	}

	if lbType, ok := d.GetOk("load_balancing_type"); ok {
		switch lbType.(string) {
		case lbRoundRobin:
			req.IPHash = 0
		case lbIPHash:
			req.IPHash = 1
		}
	}

	if geoIPMode, ok := d.GetOk("geoip_mode"); ok {
		switch geoIPMode.(string) {
		case geoIPNo:
			req.GeoIPMode = 0
		case geoIPBlockList:
			req.GeoIPMode = 1
		case geoIPAllowList:
			req.GeoIPMode = 2
		}
	}

	if geoIPList, ok := d.GetOk("geoip_list"); ok {
		iplist := geoIPList.(*schema.Set).List()
		geoIPListSet := make([]string, len(iplist))
		for i, s := range iplist {
			geoIPListSet[i] = s.(string)
		}
		req.GeoIPList = strings.Join(geoIPListSet, ",")
	}

	if redirectValue, ok := d.GetOk("www_redirect"); ok {
		if redirectValue.(bool) {
			req.WWWRedir = 1
		} else {
			req.WWWRedir = 0
		}
	}

	if waf, ok := d.GetOk("waf"); ok {
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
	config := m.(*Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, _, err := client.Resources.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", result.Name)
	d.Set("tls", result.TLSEnabled)
	d.Set("active", result.Active)
	d.Set("multiple_origins", result.MultipleOrigins)
	d.Set("wildcard_aliases", result.WidlcardAliases)
	d.Set("redirect_to_https", result.RedirectToHTTPS)
	d.Set("geoip_list", strings.Split(result.GeoIPList, ","))
	d.Set("waf", result.WAF)

	if result.HTTPS2HTTP == 1 {
		d.Set("http_to_origin", true)
	} else {
		d.Set("http_to_origin", false)
	}

	switch result.IPHash {
	case 0:
		d.Set("load_balancing_type", lbRoundRobin)
	case 1:
		d.Set("load_balancing_type", lbIPHash)
	}

	switch result.GeoIPMode {
	case 0:
		d.Set("geoip_mode", geoIPNo)
	case 1:
		d.Set("geoip_mode", geoIPBlockList)
	case 2:
		d.Set("geoip_mode", geoIPAllowList)
	}

	if result.WWWRedir == 1 {
		d.Set("www_redirect", true)
	} else {
		d.Set("www_redirect", false)
	}

	d.Set("client", result.ClientID)
	d.Set("enabled", result.Enabled)
	d.Set("ip", result.ServiceIP)
	d.Set("status", result.Status)
	d.Set("wait_for_le", result.WaitForLE)

	log.Println("[DEBUG] Finish DDoS Protection Resource reading")

	return nil
}

func resourceProtectionResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start DDoS Protection Resource updating (id=%s)\n", resourceID)
	config := m.(*Config)
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

	tlsEnabled := d.Get("tls").(*schema.Set).List()
	req.TLSEnabled = make([]string, len(tlsEnabled))
	for i, s := range tlsEnabled {
		req.TLSEnabled[i] = s.(string)
	}

	if active, ok := d.GetOk("active"); ok {
		req.Active = active.(bool)
	}

	if multipleOrigins, ok := d.GetOk("multiple_origins"); ok {
		req.MultipleOrigins = multipleOrigins.(bool)
	}

	if wildcardAliases, ok := d.GetOk("wildcard_aliases"); ok {
		req.WidlcardAliases = wildcardAliases.(bool)
	}

	if redirectToHTTPS, ok := d.GetOk("redirect_to_https"); ok {
		req.RedirectToHTTPS = redirectToHTTPS.(bool)
	}

	if httpToOriginValue, ok := d.GetOk("http_to_origin"); ok {
		if httpToOriginValue.(bool) {
			req.HTTPS2HTTP = 1
		} else {
			req.HTTPS2HTTP = 0
		}
	}

	if lbType, ok := d.GetOk("load_balancing_type"); ok {
		switch lbType.(string) {
		case lbRoundRobin:
			req.IPHash = 0
		case lbIPHash:
			req.IPHash = 1
		}
	}

	if geoIPMode, ok := d.GetOk("geoip_mode"); ok {
		switch geoIPMode.(string) {
		case geoIPNo:
			req.GeoIPMode = 0
		case geoIPBlockList:
			req.GeoIPMode = 1
		case geoIPAllowList:
			req.GeoIPMode = 2
		}
	}

	if geoIPList, ok := d.GetOk("geoip_list"); ok {
		iplist := geoIPList.(*schema.Set).List()
		geoIPListSet := make([]string, len(iplist))
		for i, s := range iplist {
			geoIPListSet[i] = s.(string)
		}
		req.GeoIPList = strings.Join(geoIPListSet, ",")
	}

	if redirectValue, ok := d.GetOk("www_redirect"); ok {
		if redirectValue.(bool) {
			req.WWWRedir = 1
		} else {
			req.WWWRedir = 0
		}
	}

	if waf, ok := d.GetOk("waf"); ok {
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
	config := m.(*Config)
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
