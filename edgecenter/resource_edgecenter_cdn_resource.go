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
				Description: "Set a list of allowed HTTP methods for the CDN content.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Allowed values are \"GET\", \"HEAD\", \"POST\", \"PUT\", \"PATCH\", \"DELETE\", and \"OPTIONS\".",
						},
					},
				},
			},
			"brotli_compression": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow compressing content with Brotli on CDN. CDN servers will request only uncompressed content from the source. It is not supported unless the Origin shielding is enabled. Brotli compression is not supported when \"fetch_compressed\" or \"slice\" are enabled.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Allowed values are \"application/javascript\", \"application/json\", \"application/vnd.ms-fontobject\", \"application/x-font-ttf\", \"application/x-javascript\", \"application/xml\", \"application/xml+rss\", \"image/svg+xml\", \"image/x-icon\", \"text/css\", \"text/html\", \"text/javascript\", \"text/plain\", \"text/xml\".",
						},
					},
				},
			},
			"browser_cache_settings": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "Set the cache lifetime for the end users’ browsers in seconds.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the cache lifetime if the CDN controlled option is chosen. If the value is empty, the Origin controlled option will be enabled and the cache lifetime will be inherited from the source. Set to \"0s\" to disable browser caching. The value only applies for the HTTP 200, 201, 204, 206, 301, 302, 303, 304, 307, 308 response status codes. Responses with other HTTP status codes will not be cached.",
						},
					},
				},
			},
			"cors": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Add the Access-Control-Allow-Origin header to responses from the CDN servers.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the value of the Access-Control-Allow-Origin header. Allowed values are \"*\", \"domain.com\" or other domain name, or \"$http_origin\".",
						},
						"always": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Add the Access-Control-Allow-Origin header to the response regardless of the HTTP response status code. Allowed values are \"true\" or \"false\". If set to \"false\", the header is only added to the responses with HTTP 200, 201, 204, 206, 301, 302, 303, 304, 307, or 308 response status codes.",
						},
					},
				},
			},
			"country_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Control access to content from the specified countries.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the policy type. Allowed values are \"allow\" or \"deny\". The policy allows or denies access to content from all countries except those specified in the \"excepted_values\" field.",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the list of countries according to ISO-3166-1.",
						},
					},
				},
			},
			"disable_proxy_force_ranges": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow CDN to get the HTTP 206 status codes regardless of the settings on the source.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"edge_cache_settings": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "Set the cache expiration time for CDN servers. The \"value\" and \"default\" fields cannot be used simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the caching time in seconds. Use the field if you want CDN to control the caching time of the HTTP 200, 206, 301, and 302 response status codes. Responses with HTTP 4xx, 5xx status codes will not be cached. Use the \"custom_values\" field to specify the custom caching time for responses with specific HTTP status codes.",
						},
						"default": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the caching time in seconds. Use the field if you want your source to control the caching time of the HTTP 200, 201, 204, 206, 301, 302, 303, 304, 307, 308 response status codes, and if a source server does not have any caching HTTP headers. Responses with other HTTP status codes will not be cached",
						},
						"custom_values": {
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
							DefaultFunc: func() (interface{}, error) {
								return map[string]interface{}{}, nil
							},
							Elem:        schema.TypeString,
							Description: "Set the caching time in seconds for certain HTTP status codes. Use \"any\" to specify the caching time for all HTTP response status codes. Use \"0s\" to disable caching.",
						},
					},
				},
			},
			"fetch_compressed": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Let CDN pull pre-compressed content from the source and cache it. Your source should support compression. The CDN servers won't ungzip your content even if a user's browser doesn't accept compression. The option is not supported when \"brotli_compression\" or \"slice\" are enabled.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"follow_origin_redirect": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "If the source returns a redirect, let CDN pull the requested content from the source that was returned in the redirect.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"codes": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeInt},
							Required:    true,
							Description: "Add the redirect HTTP status codes returned by the source. Allowed values are \"301\", \"302\", \"303\", \"307\", \"308\".",
						},
						"use_host": {
							Type:        schema.TypeBool,
							Computed:    true,
							Optional:    true,
							Description: "Use the redirect target domain as a Host header, or leave it the same as the value of the Change Host header option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"force_return": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Apply custom HTTP status codes to CDN content. Some HTTP status codes are reserved by our system and cannot be used with this option: 408, 444, 477, 494, 495, 496, 497, 499. ",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"code": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Set the HTTP status code that should be returned by the CDN. Allowed values are from \"100\" to \"599\".",
						},
						"body": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Add the URL for redirection or the text message.",
						},
					},
				},
			},
			"forward_host_header": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow forwarding the Host header used in the request made to the CDN when the CDN requests content from the source. \"host_header\" and \"forward_host_header\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"geo_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Shows the state of the Geolocation access policy option. The option controls access to content from the specified countries and their regions.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"policy_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Shows the chosen policy type. Has either \"allow\" or \"deny\" value.",
						},
						"excepted_values": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Shows the list of countries and their regions",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Two-letter country code as defined by ISO 3166-1 alpha-2 (e.g., 'US' for United States, 'RU' for Russia).",
									},
									"values": {
										Type:        schema.TypeList,
										Required:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Description: "List of region codes for the specified country, using short English names from ISO 3166-2 (e.g., 'CA' for California in 'US', 'MOW' for Moscow in 'RU').",
									},
								},
							},
						},
					},
				},
			},
			"gzip_compression": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow compressing content with gzip on CDN. CDN servers will request only uncompressed content from the source. The option is not supported when \"fetch_compressed\" or \"slice\" are enabled.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\". ",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Allowed values are \"application/dash+xml\", \"application/javascript\", \"application/javascript\", \"application/vnd.apple.mpegurl\", \"application/vnd.ms-fontobject\", \"application/wasm\", \"application/x-font-opentype\", \"application/x-font-ttf\", \"application/x-javascript\", \"application/x-mpegURL\", \"application/x-subrip\", \"application/xml\", \"application/xml+rss\", \"font/woff\", \"font/woff2\", \"image/svg+xml\", \"text/css\", \"text/html\", \"text/javascript\", \"text/plain\", \"text/vtt\", \"text/xml\".",
						},
					},
				},
			},
			"host_header": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Manage the custom Host header in the Host header option. When the CDN requests content from the source, it will use the specified Host header. \"host_header\" and \"forward_host_header\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specify the Host header value.",
						},
					},
				},
			},
			"http3_enabled": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Enable the HTTP/3 protocol for content delivery when supported by the end user’s browser.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"ignore_cookie": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Specify how to cache files with different values of the Set-Cookie header: as one object (when the option is enabled) or as different objects (when the option is disabled).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"ignore_query_string": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Specify how to cache files with different query strings: as one object (when the option is enabled) or as different objects (when the option is disabled). \"ignore_query_string\", \"query_params_whitelist\", and \"query_params_blacklist\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\". If set to \"true\", Ignore all setting is selected.",
						},
					},
				},
			},
			"image_stack": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow transforming JPG and PNG images and converting them into WebP or AVIF format.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"avif_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Allow to convert the JPG and PNG images into AVIF format when supported by the end user's browser. Allowed values are \"true\" or \"false\".",
						},
						"webp_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Allow to convert the JPG and PNG images into WebP format when supported by the end user's browser. Allowed values are \"true\" or \"false\".",
						},
						"quality": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Set the quality of the JPG and PNG images after conversion. The higher the value, the better the image quality and the larger the file size after conversion.",
						},
						"png_lossless": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Specify if the PNG images should be compressed without the quality loss.",
						},
					},
				},
			},
			"ip_address_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Control access to content from the specified IP addresses.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the policy type. Allowed values are \"allow\" or \"deny\". The policy allows or denies access to content from all IP addresses except those specified in the \"excepted_values\" field.",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the list of IP addresses.",
						},
					},
				},
			},
			"limit_bandwidth": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Control the download speed per connection.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"limit_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the speed limit type. Allowed values are \"static\" or \"dynamic\". If set to \"static\", use the \"speed\" and \"buffer\" fields. If set to \"dynamic\", the speed is limited according to the \"?speed\" and \"?buffer\" query parameters.",
						},
						"speed": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Set the maximum download speed per connection in KB/s. Must be greater than \"0\".",
						},
						"buffer": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "Specify the amount of downloaded data in KB after which the user will be rate limited.",
						},
					},
				},
			},
			"proxy_cache_methods_set": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow the caching for GET, HEAD, and POST requests.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"query_params_blacklist": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Manage the Ignore only setting of the Query string option. The setting allows CDN to ignore the specified params and cache them as one object. \"ignore_query_string\", \"query_params_whitelist\", and \"query_params_blacklist\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the list of params that should be ignored.",
						},
					},
				},
			},
			"query_params_whitelist": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Manage the Ignore all except setting of the Query string option. The setting allows CDN to ignore all but specified params and cache them as separate objects. \"ignore_query_string\", \"query_params_whitelist\", and \"query_params_blacklist\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the list of params that should not be ignored.",
						},
					},
				},
			},
			"redirect_http_to_https": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Let CDN redirect HTTPS requests to HTTP. \"redirect_http_to_https\" and \"redirect_https_to_http\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"redirect_https_to_http": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Let CDN redirect HTTP requests to HTTPS. \"redirect_http_to_https\" and \"redirect_https_to_http\" cannot be enabled simultaneously.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"referrer_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Сontrol access to content from the specified domain names.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the policy type. Allowed values are \"allow\" or \"deny\". The policy allows or denies access to content from all domain names except those specified in the \"excepted_values\" field.",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add a list of domain names. To allow a direct link access, add an empty value \"\". You cannot enter just the empty value because at least one valid referer is required.",
						},
					},
				},
			},
			"response_headers_hiding_policy": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Specify the HTTP headers set on the source that CDN servers should hide from the response.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"mode": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the way the HTTP headers are displayed. Allowed values are \"hide\" or \"show\". If set to \"hide\", all the HTTP headers from the response except those listed in the \"excepted\" field. If set to \"show\", the HTTP headers listed in the \"excepted\" field are hidden from the response.",
						},
						"excepted": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Depending on the value of the \"mode\" field, list the HTTP headers that will be either shown or hidden in the response. HTTP headers, that can't be hidden from the response: Connection, Content-Length, Content-Type, Date, Server.",
						},
					},
				},
			},
			"rewrite": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Change and redirect the requests from the CDN to the source.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"body": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Specify the rewrite pattern.",
						},
						"flag": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "break",
							Description: "Specify a rewrite flag type. Allowed values are \"last\", \"break\", \"redirect\", or \"permanent\".",
						},
					},
				},
			},
			"secure_key": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Configure access to content with tokenized URLs, generated with the MD5 algorithm.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Add the key generated on your side which will be used for the URL signing.",
						},
						"type": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Set the type of the URL signing. Allowed values are \"0\" or \"2\". If set to \"0\", the end user's IP address is inclded to secure token generation. If set to \"2\", the end user's IP address is excluded from the secure token generation.",
						},
					},
				},
			},
			"slice": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Speed up the delivery of large files and their caching. When enabled, the files are requested and cached in 10 MB chunks. The option reduces the time to first byte. The source must support the HTTP Range requests. The option is not supported when \"fetch_compressed\", \"brotli_compression\", or \"gzip_compression\" are enabled.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
						},
					},
				},
			},
			"sni": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "Help the resource understand which certificate to use for the connection, if the source server presents multiple certificates. The option works only if the \"origin_protocol\" field is set to \"HTTPS\" or \"MATCH\".",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"sni_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the SNI type. Allowed values are \"dynamic\" or \"custom\". If set to \"dynamic\", the hostname matches the value of the \"host_header\" or \"forward_host_header\" field. If set to \"custom\", the hostname matches the value of the \"custom_hostname\" field.",
						},
						"custom_hostname": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Specify the custom SNI hostname.",
						},
					},
				},
			},
			"stale": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "Let CDN serve stale cached content in case of the source unavailability.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add a list of errors. Allowed values are \"error\", \"http_403\", \"http_404\", \"http_429\", \"http_500\", \"http_502\", \"http_503\", \"http_504\", \"invalid_header\", \"timeout\", \"updating\".",
						},
					},
				},
			},
			"static_request_headers": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Let CDN add custom HTTP request headers when making requests to the source. You can specify up to 50 custom HTTP request headers. ",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeMap,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add the list of custom HTTP request headers in the \"name: value\" format. Header name is restricted to 255 symbols and can contain Latin letters (A-Z, a-z), numbers (0-9), dashes, and underscores\nHeader value is restricted to 512 symbols and must start with a letter, a number, an asterisk or {. It can contain only Latin letters (A-Z, a-z), numbers (0-9), spaces and symbols (`~!@#%^&*()-_=+ /|\";:?.><{}[]). Space can be used only between the words.",
						},
					},
				},
			},
			"static_response_headers": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Let CDN add custom HTTP response headers to the responses for the end users. You can specify up to 50 custom HTTP response headers.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Add the list of custom HTTP response headers, using the \"name\", \"value\", and \"always\" fields.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Add the header name.",
									},
									"value": {
										Type:        schema.TypeSet,
										Elem:        &schema.Schema{Type: schema.TypeString},
										Required:    true,
										Description: "Add the header value.",
									},
									"always": {
										Type:        schema.TypeBool,
										Optional:    true,
										Computed:    true,
										Description: "Specify if the custom header should be added to the responses from CDN regardless of the HTTP response status code. Allowed values are \"true\" or \"false\". If set to \"false\", the header will only be added to the responses with HTTP 200, 201, 204, 206, 301, 302, 303, 304, 307, or 308 status codes.",
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
				Description: "Specify a list of allowed SSL/TLS protocol versions. By default, all the protocol versions are allowed.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Specify the list of TLS versions. Allowed values are \"SSLv3\", \"TLSv1\", \"TLSv1.1\", \"TLSv1.2\", \"TLSv1.3\".",
						},
					},
				},
			},
			"use_default_le_chain": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Choose the Let's Encrypt certificate chain that will be used on the CDN resource. By default, the certificate is issued with the alternative chain.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Choose the certificate chain type. Allowed values are \"true\" or \"false\". If set to \"true\", a default chain is chosen. Use it for Android devices 7.1.1 or lower. If set to \"false\", an alternative certificate chain is chosen.",
						},
					},
				},
			},
			"user_agent_acl": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Control access to content for the specified user agents.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"policy_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Set the policy type. Allowed values are \"allow\" or \"deny\". The policy allows or denies access to content from all user agents except those specified in the \"excepted_values\" field.",
						},
						"excepted_values": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Required:    true,
							Description: "Add a list of user agents. Enter the values in [\"\"]. You can specify a user agent string, an empty value using \"\", or a regular expression that starts with \"~\".",
						},
					},
				},
			},
			"websockets": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Allow WebSockets connections to the source. The WebSockets option can only be enabled upon request. Please contact support for assistance with activation.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Enable or disable the option. Allowed values are \"true\" or \"false\".",
						},
						"value": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Set the value of the option. Allowed values are \"true\" or \"false\".",
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
				Description: "Specify a custom domain that will be used to deliver content via CDN.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Leave an optional comment that describes this CDN resource.",
			},
			"origin_group": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ExactlyOneOf: []string{
					"origin_group",
					"origin",
				},
				Description: "Enter the source group ID. Use one of your source groups or create a new one. You can use either \"origin\" or \"origin_group\" field in the request.",
			},
			"origin": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ExactlyOneOf: []string{
					"origin_group",
					"origin",
				},
				Description: "Enter a domain name or the IP address of your source. Specify a port if custom. You can use either \"origin\" or \"origin_group\" field in the request.",
			},
			"origin_protocol": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Choose the protocol that will be used by CDN servers to request content from the source. If not specified, the HTTP protocol will be used. Allowed values are \"HTTPS\", \"HTTP\", or \"MATCH\". If \"MATCH\" is chosen, content on the source should be available over both HTTP and HTTPS protocols.",
			},
			"secondary_hostnames": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				DefaultFunc: func() (interface{}, error) {
					return []string{}, nil
				},
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Specify up to 10 additional custom domains that will be used to deliver content via the CDN.",
			},
			"ssl_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable or disable the HTTPS protocol for content delivery.",
			},
			"ssl_data": {
				Type:         schema.TypeInt,
				Optional:     true,
				RequiredWith: []string{"ssl_enabled"},
				Description:  "Specify the SSL certificate ID which will be used for the CDN resource. The field must be used only with \"ssl_enabled\": true.",
			},
			"ssl_automated": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
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
				Description: "Enable or disable the CDN resource.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CDN resource status. Allowed values are \"active\", \"suspended\", or \"processed\".",
			},
			"ssl_le_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "",
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

	uniqHostnames := make(map[string]struct{})
	for _, hostname := range d.Get("secondary_hostnames").(*schema.Set).List() {
		hostStr := hostname.(string)
		if _, ok := uniqHostnames[hostStr]; !ok {
			uniqHostnames[hostStr] = struct{}{}
			req.SecondaryHostnames = append(req.SecondaryHostnames, hostStr)
		}
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
	d.Set("ssl_le_enabled", result.SSLLEEnabled)
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
	req.SSLAutomated = d.Get("ssl_automated").(bool)
	req.OriginProtocol = resources.Protocol(d.Get("origin_protocol").(string))
	req.Options = listToResourceOptions(d.Get("options").([]interface{}))

	uniqHostnames := make(map[string]struct{})
	for _, hostname := range d.Get("secondary_hostnames").(*schema.Set).List() {
		hostStr := hostname.(string)
		if _, ok := uniqHostnames[hostStr]; !ok {
			uniqHostnames[hostStr] = struct{}{}
			req.SecondaryHostnames = append(req.SecondaryHostnames, hostStr)
		}
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
			UseHost: opt["use_host"].(bool),
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
	if opt, ok := getOptByName(fields, "geo_acl"); ok {
		opts.GeoAcl = &cdn.GeoAccessPolicy{
			Enabled:  opt["enabled"].(bool),
			Default:  opt["policy_type"].(string),
			Excepted: map[string][]string{},
		}
		if exceptList, ok := opt["excepted_values"].([]interface{}); ok {
			for _, item := range exceptList {
				except := item.(map[string]interface{})
				key := except["key"].(string)
				values := except["values"].([]interface{})
				strValues := make([]string, len(values))
				for i, val := range values {
					strValues[i] = val.(string)
				}
				opts.GeoAcl.Excepted[key] = strValues
			}
		}
	}
	if opt, ok := getOptByName(fields, "gzip_compression"); ok {
		opts.GzipCompression = &cdn.GzipCompression{
			Enabled: opt["enabled"].(bool),
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.GzipCompression.Value = append(opts.GzipCompression.Value, v.(string))
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
	if options.GeoAcl != nil {
		var exceptedValues []interface{}
		for key, values := range options.GeoAcl.Excepted {
			exceptedValues = append(exceptedValues, map[string]interface{}{
				"key":    key,
				"values": values,
			})
		}
		result["geo_acl"] = []interface{}{
			map[string]interface{}{
				"enabled":         options.GeoAcl.Enabled,
				"policy_type":     options.GeoAcl.Default,
				"excepted_values": exceptedValues,
			},
		}
	}
	if options.GzipCompression != nil {
		m := structToMap(options.GzipCompression)
		result["gzip_compression"] = []interface{}{m}
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
