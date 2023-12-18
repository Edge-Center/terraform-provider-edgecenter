package edgecenter

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cdn "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/resources"
)

var resourceOptionsSchema = &schema.Schema{
	Type:        schema.TypeList,
	MaxItems:    1,
	Optional:    true,
	Computed:    true,
	Description: "Each option in CDN resource settings. Each option added to CDN resource settings should have the following mandatory request fields: enabled, value.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"allowed_http_methods": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"brotli_compression": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"browser_cache_settings": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "",
						},
					},
				},
			},
			"cache_http_headers": { // Deprecated. Use - response_headers_hiding_policy.
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"cors": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
						"always": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
					},
				},
			},
			"country_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"disable_proxy_force_ranges": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"edge_cache_settings": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "The cache expiration time for CDN servers.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Caching time for a response with codes 200, 206, 301, 302. Responses with codes 4xx, 5xx will not be cached. Use '0s' disable to caching. Use custom_values field to specify a custom caching time for a response with specific codes.",
						},
						"default": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Content will be cached according to origin cache settings. The value applies for a response with codes 200, 201, 204, 206, 301, 302, 303, 304, 307, 308 if an origin server does not have caching HTTP headers. Responses with other codes will not be cached.",
						},
						"custom_values": {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return map[string]interface{}{}, nil
							},
							Elem:        schema.TypeString,
							Description: "Caching time for a response with specific codes. These settings have a higher priority than the value field. Response code ('304', '404' for example). Use 'any' to specify caching time for all response codes. Caching time in seconds ('0s', '600s' for example). Use '0s' to disable caching for a specific response code.",
						},
					},
				},
			},
			"fetch_compressed": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"follow_origin_redirect": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"codes": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeInt},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"force_return": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"code": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "",
						},
						"body": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "",
						},
					},
				},
			},
			"forward_host_header": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"gzip_on": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"host_header": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Specify the Host header that CDN servers use when request content from an origin server. Your server must be able to process requests with the chosen header. If the option is in NULL state Host Header value is taken from the CNAME field.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"http3_enabled": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"ignore_cookie": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"ignore_query_string": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"image_stack": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"avif_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
						"webp_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
						"quality": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "",
						},
						"png_lossless": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
					},
				},
			},
			"ip_address_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"limit_bandwidth": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"limit_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"speed": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
						"buffer": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "",
						},
					},
				},
			},
			"proxy_cache_methods_set": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"query_params_blacklist": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeSet,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Required: true,
						},
					},
				},
			},
			"query_params_whitelist": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeSet,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Required: true,
						},
					},
				},
			},
			"redirect_http_to_https": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Sets redirect from HTTP protocol to HTTPS for all resource requests.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"redirect_https_to_http": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"referrer_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Possible values: allow, deny.",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"response_headers_hiding_policy": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"mode": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"excepted": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"rewrite": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"body": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"flag": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "break",
							Description: "",
						},
					},
				},
			},
			"secure_key": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"type": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"slice": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"sni": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"sni_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Available values 'dynamic' or 'custom'",
						},
						"custom_hostname": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Required to set custom hostname in case sni-type='custom'",
						},
					},
				},
			},
			"stale": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"static_headers": { // Deprecated. Use - static_response_headers.
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Option has been deprecated. Use - static_response_headers.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeMap,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Required: true,
						},
					},
				},
			},
			"static_request_headers": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeMap,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"static_response_headers": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Specify custom HTTP Headers that a CDN server adds to a response.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "",
									},
									"value": {
										Type:        schema.TypeSet,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Required:    true,
										Description: "",
									},
									"always": {
										Type:        schema.TypeBool,
										Optional:    true,
										Computed:    true,
										Description: "",
									},
								},
							},
						},
					},
				},
			},
			"tls_versions": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"use_default_le_chain": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"user_agent_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "",
						},
					},
				},
			},
			"websockets": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"value": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
		},
	},
}

func resourceCDNResource() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"cname": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A CNAME that will be used to deliver content though a CDN. If you update this field new resource will be created.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom client description of the resource.",
			},
			"origin_group": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ExactlyOneOf: []string{
					"origin_group",
					"origin",
				},
				Description: "ID of the Origins Group. Use one of your Origins Group or create a new one. You can use either 'origin' parameter or 'originGroup' in the resource definition.",
			},
			"origin": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ExactlyOneOf: []string{
					"origin_group",
					"origin",
				},
				Description: "A domain name or IP of your origin source. Specify a port if custom. You can use either 'origin' parameter or 'originGroup' in the resource definition.",
			},
			"origin_protocol": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "This option defines the protocol that will be used by CDN servers to request content from an origin source. If not specified, we will use HTTP to connect to an origin server. Possible values are: HTTPS, HTTP, MATCH.",
			},
			"secondary_hostnames": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				DefaultFunc: func() (interface{}, error) {
					return []string{}, nil
				},
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of additional CNAMEs.",
			},
			"ssl_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Use HTTPS protocol for content delivery.",
			},
			"ssl_data": {
				Type:         schema.TypeInt,
				Optional:     true,
				RequiredWith: []string{"ssl_enabled"},
				Description:  "Specify the SSL Certificate ID which should be used for the CDN Resource.",
			},
			"ssl_automated": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "generate LE certificate automatically.",
			},
			"issue_le_cert": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Generate LE certificate.",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "The setting allows to enable or disable a CDN Resource",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of a CDN resource content availability. Possible values are: Active, Suspended, Processed.",
			},
			"options": resourceOptionsSchema,
		},
		CreateContext: resourceCDNResourceCreate,
		ReadContext:   resourceCDNResourceRead,
		UpdateContext: resourceCDNResourceUpdate,
		DeleteContext: resourceCDNResourceDelete,
		Description:   "Represent CDN resource",
	}
}

func resourceCDNResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start CDN Resource creating")
	config := m.(*Config)
	client := config.CDNClient

	var req resources.CreateRequest
	req.Cname = d.Get("cname").(string)
	req.Description = d.Get("description").(string)
	req.Origin = d.Get("origin").(string)
	req.OriginGroup = d.Get("origin_group").(int)
	req.OriginProtocol = resources.Protocol(d.Get("origin_protocol").(string))
	req.SSlEnabled = d.Get("ssl_enabled").(bool)
	req.SSLData = d.Get("ssl_data").(int)
	req.SSLAutomated = d.Get("ssl_automated").(bool)

	if d.Get("issue_le_cert") != nil {
		req.IssueLECert = d.Get("issue_le_cert").(bool)
	}

	req.Options = listToResourceOptions(d.Get("options").([]interface{}))

	for _, hostname := range d.Get("secondary_hostnames").(*schema.Set).List() {
		req.SecondaryHostnames = append(req.SecondaryHostnames, hostname.(string))
	}

	result, err := client.Resources().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	resourceCDNResourceRead(ctx, d, m)

	log.Printf("[DEBUG] Finish CDN Resource creating (id=%d)\n", result.ID)

	return nil
}

func resourceCDNResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start CDN Resource reading (id=%s)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, err := client.Resources().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("cname", result.Cname)
	d.Set("description", result.Description)
	d.Set("origin_group", result.OriginGroup)
	d.Set("origin_protocol", result.OriginProtocol)
	d.Set("secondary_hostnames", result.SecondaryHostnames)
	d.Set("ssl_enabled", result.SSlEnabled)
	d.Set("ssl_data", result.SSLData)
	d.Set("ssl_automated", result.SSLAutomated)
	d.Set("status", result.Status)
	d.Set("active", result.Active)
	if err := d.Set("options", resourceOptionsToList(result.Options)); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish CDN Resource reading")

	return nil
}

func resourceCDNResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start CDN Resource updating (id=%s)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req resources.UpdateRequest
	req.Active = d.Get("active").(bool)
	req.Description = d.Get("description").(string)
	req.OriginGroup = d.Get("origin_group").(int)
	req.SSlEnabled = d.Get("ssl_enabled").(bool)
	req.SSLData = d.Get("ssl_data").(int)
	req.OriginProtocol = resources.Protocol(d.Get("origin_protocol").(string))
	req.Options = listToResourceOptions(d.Get("options").([]interface{}))
	for _, hostname := range d.Get("secondary_hostnames").(*schema.Set).List() {
		req.SecondaryHostnames = append(req.SecondaryHostnames, hostname.(string))
	}

	if _, err := client.Resources().Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish CDN Resource updating")

	return resourceCDNResourceRead(ctx, d, m)
}

func resourceCDNResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start CDN Resource deleting (id=%s)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.Resources().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish CDN Resource deleting")

	return nil
}

func listToResourceOptions(l []interface{}) *cdn.ResourceOptions {
	if len(l) == 0 {
		return nil
	}

	var opts cdn.ResourceOptions
	fields := l[0].(map[string]interface{})
	if opt, ok := getOptByName(fields, "allowed_http_methods"); ok {
		opts.AllowedHTTPMethods = &cdn.AllowedHTTPMethods{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.AllowedHTTPMethods.Value = append(opts.AllowedHTTPMethods.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "brotli_compression"); ok {
		opts.BrotliCompression = &cdn.BrotliCompression{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.BrotliCompression.Value = append(opts.BrotliCompression.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "browser_cache_settings"); ok {
		opts.BrowserCacheSettings = &cdn.BrowserCacheSettings{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "cache_http_headers"); ok {
		opts.CacheHttpHeaders = &cdn.CacheHttpHeaders{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.CacheHttpHeaders.Value = append(opts.CacheHttpHeaders.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "cors"); ok {
		opts.Cors = &cdn.Cors{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.Cors.Value = append(opts.Cors.Value, v.(string))
		}
		if _, ok := opt["always"]; ok {
			opts.Cors.Always = opt["always"].(bool)
		}
	}
	if opt, ok := getOptByName(fields, "country_acl"); ok {
		opts.CountryACL = &cdn.CountryACL{
			Enabled:    opt["enabled"].(bool),
			PolicyType: opt["policy_type"].(string),
		}
		for _, v := range opt["excepted_values"].(*schema.Set).List() {
			opts.CountryACL.ExceptedValues = append(opts.CountryACL.ExceptedValues, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "disable_proxy_force_ranges"); ok {
		opts.DisableProxyForceRanges = &cdn.DisableProxyForceRanges{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "edge_cache_settings"); ok {
		rawCustomVals := opt["custom_values"].(map[string]interface{})
		customVals := make(map[string]string, len(rawCustomVals))
		for key, value := range rawCustomVals {
			customVals[key] = value.(string)
		}

		opts.EdgeCacheSettings = &cdn.EdgeCacheSettings{
			Enabled:      opt["enabled"].(bool),
			Value:        opt["value"].(string),
			CustomValues: customVals,
			Default:      opt["default"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "fetch_compressed"); ok {
		opts.FetchCompressed = &cdn.FetchCompressed{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "follow_origin_redirect"); ok {
		opts.FollowOriginRedirect = &cdn.FollowOriginRedirect{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["codes"].(*schema.Set).List() {
			opts.FollowOriginRedirect.Codes = append(opts.FollowOriginRedirect.Codes, v.(int))
		}
	}
	if opt, ok := getOptByName(fields, "force_return"); ok {
		opts.ForceReturn = &cdn.ForceReturn{
			Enabled: opt["enabled"].(bool),
			Code:    opt["code"].(int),
			Body:    opt["body"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "forward_host_header"); ok {
		opts.ForwardHostHeader = &cdn.ForwardHostHeader{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "gzip_on"); ok {
		opts.GzipOn = &cdn.GzipOn{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "host_header"); ok {
		opts.HostHeader = &cdn.HostHeader{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "http3_enabled"); ok {
		opts.HTTP3Enabled = &cdn.HTTP3Enabled{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "ignore_cookie"); ok {
		opts.IgnoreCookie = &cdn.IgnoreCookie{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "ignore_query_string"); ok {
		opts.IgnoreQueryString = &cdn.IgnoreQueryString{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "image_stack"); ok {
		opts.ImageStack = &cdn.ImageStack{
			Enabled: opt["enabled"].(bool),
			Quality: opt["quality"].(int),
		}
		if _, ok := opt["avif_enabled"]; ok {
			opts.ImageStack.AvifEnabled = opt["avif_enabled"].(bool)
		}
		if _, ok := opt["webp_enabled"]; ok {
			opts.ImageStack.WebpEnabled = opt["webp_enabled"].(bool)
		}
		if _, ok := opt["png_lossless"]; ok {
			opts.ImageStack.PngLossless = opt["png_lossless"].(bool)
		}
	}
	if opt, ok := getOptByName(fields, "ip_address_acl"); ok {
		opts.IPAddressACL = &cdn.IPAddressACL{
			Enabled:    opt["enabled"].(bool),
			PolicyType: opt["policy_type"].(string),
		}
		for _, v := range opt["excepted_values"].(*schema.Set).List() {
			opts.IPAddressACL.ExceptedValues = append(opts.IPAddressACL.ExceptedValues, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "limit_bandwidth"); ok {
		opts.LimitBandwidth = &cdn.LimitBandwidth{
			Enabled:   opt["enabled"].(bool),
			LimitType: opt["limit_type"].(string),
		}
		if _, ok := opt["speed"]; ok {
			opts.LimitBandwidth.Speed = opt["speed"].(int)
		}
		if _, ok := opt["buffer"]; ok {
			opts.LimitBandwidth.Buffer = opt["buffer"].(int)
		}
	}
	if opt, ok := getOptByName(fields, "proxy_cache_methods_set"); ok {
		opts.ProxyCacheMethodsSet = &cdn.ProxyCacheMethodsSet{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "query_params_blacklist"); ok {
		opts.QueryParamsBlacklist = &cdn.QueryParamsBlacklist{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.QueryParamsBlacklist.Value = append(opts.QueryParamsBlacklist.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "query_params_whitelist"); ok {
		opts.QueryParamsWhitelist = &cdn.QueryParamsWhitelist{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.QueryParamsWhitelist.Value = append(opts.QueryParamsWhitelist.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "redirect_http_to_https"); ok {
		opts.RedirectHttpToHttps = &cdn.RedirectHttpToHttps{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "redirect_https_to_http"); ok {
		opts.RedirectHttpsToHttp = &cdn.RedirectHttpsToHttp{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "referrer_acl"); ok {
		opts.ReferrerACL = &cdn.ReferrerACL{
			Enabled:    opt["enabled"].(bool),
			PolicyType: opt["policy_type"].(string),
		}
		for _, v := range opt["excepted_values"].(*schema.Set).List() {
			opts.ReferrerACL.ExceptedValues = append(opts.ReferrerACL.ExceptedValues, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "response_headers_hiding_policy"); ok {
		opts.ResponseHeadersHidingPolicy = &cdn.ResponseHeadersHidingPolicy{
			Enabled: opt["enabled"].(bool),
			Mode:    opt["mode"].(string),
		}
		for _, v := range opt["excepted"].(*schema.Set).List() {
			opts.ResponseHeadersHidingPolicy.Excepted = append(opts.ResponseHeadersHidingPolicy.Excepted, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "rewrite"); ok {
		opts.Rewrite = &cdn.Rewrite{
			Enabled: opt["enabled"].(bool),
			Body:    opt["body"].(string),
			Flag:    opt["flag"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "secure_key"); ok {
		opts.SecureKey = &cdn.SecureKey{
			Enabled: opt["enabled"].(bool),
			Key:     opt["key"].(string),
			Type:    opt["type"].(int),
		}
	}
	if opt, ok := getOptByName(fields, "slice"); ok {
		opts.Slice = &cdn.Slice{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "sni"); ok {
		opts.SNI = &cdn.SNIOption{
			Enabled:        opt["enabled"].(bool),
			SNIType:        opt["sni_type"].(string),
			CustomHostname: opt["custom_hostname"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "stale"); ok {
		opts.Stale = &cdn.Stale{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.Stale.Value = append(opts.Stale.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "static_headers"); ok {
		opts.StaticHeaders = &cdn.StaticHeaders{
			Enabled: opt["enabled"].(bool),
			Value:   map[string]string{},
		}
		for k, v := range opt["value"].(map[string]interface{}) {
			opts.StaticHeaders.Value[k] = v.(string)
		}
	}
	if opt, ok := getOptByName(fields, "static_request_headers"); ok {
		opts.StaticRequestHeaders = &cdn.StaticRequestHeaders{
			Enabled: opt["enabled"].(bool),
			Value:   map[string]string{},
		}
		for k, v := range opt["value"].(map[string]interface{}) {
			opts.StaticRequestHeaders.Value[k] = v.(string)
		}
	}
	if opt, ok := getOptByName(fields, "static_response_headers"); ok {
		opts.StaticResponseHeaders = &cdn.StaticResponseHeaders{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].([]interface{}) {
			itemData := v.(map[string]interface{})
			item := &cdn.StaticResponseHeadersItem{
				Name: itemData["name"].(string),
			}
			for _, val := range itemData["value"].(*schema.Set).List() {
				item.Value = append(item.Value, val.(string))
			}
			if _, ok := itemData["always"]; ok {
				item.Always = itemData["always"].(bool)
			}
			opts.StaticResponseHeaders.Value = append(opts.StaticResponseHeaders.Value, *item)
		}
	}
	if opt, ok := getOptByName(fields, "tls_versions"); ok {
		opts.TLSVersions = &cdn.TLSVersions{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.TLSVersions.Value = append(opts.TLSVersions.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "use_default_le_chain"); ok {
		opts.UseDefaultLEChain = &cdn.UseDefaultLEChain{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "user_agent_acl"); ok {
		opts.UserAgentACL = &cdn.UserAgentACL{
			Enabled:    opt["enabled"].(bool),
			PolicyType: opt["policy_type"].(string),
		}
		for _, v := range opt["excepted_values"].(*schema.Set).List() {
			opts.UserAgentACL.ExceptedValues = append(opts.UserAgentACL.ExceptedValues, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "websockets"); ok {
		opts.WebSockets = &cdn.WebSockets{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(bool),
		}
	}

	return &opts
}

func getOptByName(fields map[string]interface{}, name string) (map[string]interface{}, bool) {
	if _, ok := fields[name]; !ok {
		return nil, false
	}

	container, ok := fields[name].([]interface{})
	if !ok {
		return nil, false
	}

	if len(container) == 0 {
		return nil, false
	}

	opt, ok := container[0].(map[string]interface{})
	if !ok {
		return nil, false
	}

	return opt, true
}

func resourceOptionsToList(options *cdn.ResourceOptions) []interface{} {
	result := make(map[string][]interface{})
	if options.AllowedHTTPMethods != nil {
		m := structToMap(options.AllowedHTTPMethods)
		result["allowed_http_methods"] = []interface{}{m}
	}
	if options.BrotliCompression != nil {
		m := structToMap(options.BrotliCompression)
		result["brotli_compression"] = []interface{}{m}
	}
	if options.BrowserCacheSettings != nil {
		m := structToMap(options.BrowserCacheSettings)
		result["browser_cache_settings"] = []interface{}{m}
	}
	if options.CacheHttpHeaders != nil {
		m := structToMap(options.CacheHttpHeaders)
		result["cache_http_headers"] = []interface{}{m}
	}
	if options.Cors != nil {
		m := structToMap(options.Cors)
		result["cors"] = []interface{}{m}
	}
	if options.CountryACL != nil {
		m := structToMap(options.CountryACL)
		result["country_acl"] = []interface{}{m}
	}
	if options.DisableProxyForceRanges != nil {
		m := structToMap(options.DisableProxyForceRanges)
		result["disable_proxy_force_ranges"] = []interface{}{m}
	}
	if options.EdgeCacheSettings != nil {
		m := structToMap(options.EdgeCacheSettings)
		result["edge_cache_settings"] = []interface{}{m}
	}
	if options.FetchCompressed != nil {
		m := structToMap(options.FetchCompressed)
		result["fetch_compressed"] = []interface{}{m}
	}
	if options.FollowOriginRedirect != nil {
		m := structToMap(options.FollowOriginRedirect)
		result["follow_origin_redirect"] = []interface{}{m}
	}
	if options.ForceReturn != nil {
		m := structToMap(options.ForceReturn)
		result["force_return"] = []interface{}{m}
	}
	if options.ForwardHostHeader != nil {
		m := structToMap(options.ForwardHostHeader)
		result["forward_host_header"] = []interface{}{m}
	}
	if options.GzipOn != nil {
		m := structToMap(options.GzipOn)
		result["gzip_on"] = []interface{}{m}
	}
	if options.HostHeader != nil {
		m := structToMap(options.HostHeader)
		result["host_header"] = []interface{}{m}
	}
	if options.HTTP3Enabled != nil {
		m := structToMap(options.HTTP3Enabled)
		result["http3_enabled"] = []interface{}{m}
	}
	if options.IgnoreCookie != nil {
		m := structToMap(options.IgnoreCookie)
		result["ignore_cookie"] = []interface{}{m}
	}
	if options.IgnoreQueryString != nil {
		m := structToMap(options.IgnoreQueryString)
		result["ignore_query_string"] = []interface{}{m}
	}
	if options.ImageStack != nil {
		m := structToMap(options.ImageStack)
		result["image_stack"] = []interface{}{m}
	}
	if options.IPAddressACL != nil {
		m := structToMap(options.IPAddressACL)
		result["ip_address_acl"] = []interface{}{m}
	}
	if options.LimitBandwidth != nil {
		m := structToMap(options.LimitBandwidth)
		result["limit_bandwidth"] = []interface{}{m}
	}
	if options.ProxyCacheMethodsSet != nil {
		m := structToMap(options.ProxyCacheMethodsSet)
		result["proxy_cache_methods_set"] = []interface{}{m}
	}
	if options.QueryParamsBlacklist != nil {
		m := structToMap(options.QueryParamsBlacklist)
		result["query_params_blacklist"] = []interface{}{m}
	}
	if options.QueryParamsWhitelist != nil {
		m := structToMap(options.QueryParamsWhitelist)
		result["query_params_whitelist"] = []interface{}{m}
	}
	if options.RedirectHttpsToHttp != nil {
		m := structToMap(options.RedirectHttpsToHttp)
		result["redirect_https_to_http"] = []interface{}{m}
	}
	if options.RedirectHttpToHttps != nil {
		m := structToMap(options.RedirectHttpToHttps)
		result["redirect_http_to_https"] = []interface{}{m}
	}
	if options.ReferrerACL != nil {
		m := structToMap(options.ReferrerACL)
		result["referrer_acl"] = []interface{}{m}
	}
	if options.ResponseHeadersHidingPolicy != nil {
		m := structToMap(options.ResponseHeadersHidingPolicy)
		result["response_headers_hiding_policy"] = []interface{}{m}
	}
	if options.Rewrite != nil {
		m := structToMap(options.Rewrite)
		result["rewrite"] = []interface{}{m}
	}
	if options.SecureKey != nil {
		m := structToMap(options.SecureKey)
		result["secure_key"] = []interface{}{m}
	}
	if options.Slice != nil {
		m := structToMap(options.Slice)
		result["slice"] = []interface{}{m}
	}
	if options.SNI != nil {
		m := structToMap(options.SNI)
		result["sni"] = []interface{}{m}
	}
	if options.Stale != nil {
		m := structToMap(options.Stale)
		result["stale"] = []interface{}{m}
	}
	if options.StaticHeaders != nil {
		m := structToMap(options.StaticHeaders)
		result["static_headers"] = []interface{}{m}
	}
	if options.StaticRequestHeaders != nil {
		m := structToMap(options.StaticRequestHeaders)
		result["static_request_headers"] = []interface{}{m}
	}
	if options.StaticResponseHeaders != nil {
		m := structToMap(options.StaticResponseHeaders)
		items := []interface{}{}
		for _, v := range m["value"].([]cdn.StaticResponseHeadersItem) {
			items = append(items, structToMap(v))
		}
		m["value"] = items
		result["static_response_headers"] = []interface{}{m}
	}
	if options.TLSVersions != nil {
		m := structToMap(options.TLSVersions)
		result["tls_versions"] = []interface{}{m}
	}
	if options.UseDefaultLEChain != nil {
		m := structToMap(options.UseDefaultLEChain)
		result["use_default_le_chain"] = []interface{}{m}
	}
	if options.UserAgentACL != nil {
		m := structToMap(options.UserAgentACL)
		result["user_agent_acl"] = []interface{}{m}
	}
	if options.WebSockets != nil {
		m := structToMap(options.WebSockets)
		result["websockets"] = []interface{}{m}
	}

	return []interface{}{result}
}

func structToMap(item interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	if item == nil {
		return res
	}
	v := reflect.TypeOf(item)
	reflectValue := reflect.ValueOf(item)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i).Interface()
		if tag != "" && tag != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[tag] = structToMap(field)
			} else {
				res[tag] = field
			}
		}
	}

	return res
}
