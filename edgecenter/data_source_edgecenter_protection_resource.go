package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProtectionResource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProtectionResourceRead,
		Description: "Represent DDoS protection resource.",
		Schema: map[string]*schema.Schema{
			"active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable DDoS protection resource.",
			},
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
			"geoip_list": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "List of countries to apply geoip_mode policy to.",
			},
			"geoip_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Manage country access policy to control access to DDoS resource from the specified countries. Available values are `%s`, `%s`, `%s`.", geoIPNo, geoIPAllowList, geoIPBlockList),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case geoIPNo, geoIPAllowList, geoIPBlockList:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`, `%s`.", v, geoIPNo, geoIPAllowList, geoIPBlockList)
				},
			},
			"http_to_origin": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether to use HTTP to make requests to the origin. If set to false (default), HTTPS is used.",
			},
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: ".",
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Resources's protected IP address.",
			},
			"load_balancing_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Sets load balancing type. Available values are `%s`, `%s`.", lbRoundRobin, lbIPHash),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case lbRoundRobin, lbIPHash:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`.", v, lbRoundRobin, lbIPHash)
				},
			},
			"multiple_origins": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable Multiple origins feature.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The DNS name of the DDoS protection resource.",
			},
			"redirect_to_https": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable from HTTP to HTTPS",
			},
			"ssl_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Select the SSL certificate type. Available values are `%s`, `%s`.", sslCustom, sslLE),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case sslCustom, sslLE:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`.", v, sslCustom, sslLE)
				},
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Show resource status.",
			},
			"tls": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: fmt.Sprintf("The list of supported TLS versions. Available value: `%s`, `%s`, `%s`, `%s`.", tlsv1, tlsv1_1, tlsv1_2, tlsv1_3),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case tlsv1, tlsv1_1, tlsv1_2, tlsv1_3:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`.", v, tlsv1, tlsv1_1, tlsv1_2, tlsv1_3)
				},
			},
			"wildcard_aliases": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable Wildcard aliases feature.",
			},
			"waf": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable WAF.",
			},
			"www_redirect": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable or disable redirect from WWW to the primary domain option.",
			},
			"wait_for_le": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of seconds after which LE certificate can be issued.",
			},
		},
	}
}

func dataSourceProtectionResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start Protection Resource reading (id=%s)\n", resourceID)
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
	d.Set("geoip_list", result.GeoIPList)
	d.Set("ssl_type", result.SSLType)
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

	log.Println("[DEBUG] Finish Protection Resource reading")

	return nil
}
