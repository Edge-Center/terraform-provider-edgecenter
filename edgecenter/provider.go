package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	dnssdk "github.com/bioidiad/edgecenter-dns-sdk-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	eccdnProvider "github.com/Edge-Center/edgecentercdn-go/edgecenter/provider"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	ec "github.com/Edge-Center/edgecentercloud-go/edgecenter"
)

const (
	ProviderOptPermanentToken    = "permanent_api_token"
	ProviderOptSkipCredsAuthErr  = "ignore_creds_auth_error" // nolint: gosec
	ProviderOptSingleAPIEndpoint = "api_endpoint"

	LifecyclePolicyResource = "edgecenter_lifecyclepolicy"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user_name": {
				Type:     schema.TypeString,
				Optional: true,
				// commented because it's broke all tests
				// AtLeastOneOf: []string{ProviderOptPermanentToken, "user_name"},
				// RequiredWith: []string{"user_name", "password"},
				Deprecated:  fmt.Sprintf("Use %s instead", ProviderOptPermanentToken),
				DefaultFunc: schema.EnvDefaultFunc("EC_USERNAME", nil),
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
				// commented because it's broke all tests
				// RequiredWith: []string{"user_name", "password"},
				Deprecated:  fmt.Sprintf("Use %s instead", ProviderOptPermanentToken),
				DefaultFunc: schema.EnvDefaultFunc("EC_PASSWORD", nil),
			},
			ProviderOptPermanentToken: {
				Type:     schema.TypeString,
				Optional: true,
				// commented because it's broke all tests
				// AtLeastOneOf: []string{ProviderOptPermanentToken, "user_name"},
				Sensitive:   true,
				Description: "A permanent [API-token](https://support.edgecenter.ru/knowledge_base/item/257788)",
				DefaultFunc: schema.EnvDefaultFunc("EC_PERMANENT_TOKEN", nil),
			},
			ProviderOptSingleAPIEndpoint: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A single API endpoint for all products. Will be used when specific product API url is not defined.",
				DefaultFunc: schema.EnvDefaultFunc("EC_API_ENDPOINT", "https://api.edgecenter.ru"),
			},
			ProviderOptSkipCredsAuthErr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Deprecated:  "It doesn't make any effect anymore",
				Description: "Should be set to true when you are gonna to use storage resource with permanent API-token only.",
			},
			"edgecenter_platform": {
				Type:          schema.TypeString,
				Optional:      true,
				Deprecated:    "Use edgecenter_platform_api instead",
				ConflictsWith: []string{"edgecenter_platform_api"},
				Description:   "Platform URL is used for generate JWT",
				DefaultFunc:   schema.EnvDefaultFunc("EC_PLATFORM", nil),
			},
			"edgecenter_platform_api": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Platform URL is used for generate JWT (define only if you want to override Platform API endpoint)",
				DefaultFunc: schema.EnvDefaultFunc("EC_PLATFORM_API", nil),
			},
			"edgecenter_api": {
				Type:          schema.TypeString,
				Optional:      true,
				Deprecated:    "Use edgecenter_cloud_api instead",
				ConflictsWith: []string{"edgecenter_cloud_api"},
				Description:   "Region API",
				DefaultFunc:   schema.EnvDefaultFunc("EC_API", nil),
			},
			"edgecenter_cloud_api": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Region API (define only if you want to override Region API endpoint)",
				DefaultFunc: schema.EnvDefaultFunc("EC_CLOUD_API", nil),
			},
			"edgecenter_cdn_api": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "CDN API (define only if you want to override CDN API endpoint)",
				DefaultFunc: schema.EnvDefaultFunc("EC_CDN_API", ""),
			},
			"edgecenter_storage_api": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Storage API (define only if you want to override Storage API endpoint)",
				DefaultFunc: schema.EnvDefaultFunc("EC_STORAGE_API", ""),
			},
			"edgecenter_dns_api": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DNS API (define only if you want to override DNS API endpoint)",
				DefaultFunc: schema.EnvDefaultFunc("EC_DNS_API", ""),
			},
			"edgecenter_client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Client id",
				DefaultFunc: schema.EnvDefaultFunc("EC_CLIENT_ID", ""),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"edgecenter_volume":            resourceVolume(),
			"edgecenter_network":           resourceNetwork(),
			"edgecenter_subnet":            resourceSubnet(),
			"edgecenter_router":            resourceRouter(),
			"edgecenter_instance":          resourceInstance(),
			"edgecenter_keypair":           resourceKeypair(),
			"edgecenter_reservedfixedip":   resourceReservedFixedIP(),
			"edgecenter_floatingip":        resourceFloatingIP(),
			"edgecenter_loadbalancer":      resourceLoadBalancer(),
			"edgecenter_loadbalancerv2":    resourceLoadBalancerV2(),
			"edgecenter_lblistener":        resourceLbListener(),
			"edgecenter_lbpool":            resourceLBPool(),
			"edgecenter_lbmember":          resourceLBMember(),
			"edgecenter_securitygroup":     resourceSecurityGroup(),
			"edgecenter_baremetal":         resourceBmInstance(),
			"edgecenter_snapshot":          resourceSnapshot(),
			"edgecenter_servergroup":       resourceServerGroup(),
			"edgecenter_k8s":               resourceK8s(),
			"edgecenter_k8s_pool":          resourceK8sPool(),
			"edgecenter_secret":            resourceSecret(),
			"edgecenter_storage_s3":        resourceStorageS3(),
			"edgecenter_storage_s3_bucket": resourceStorageS3Bucket(),
			DNSZoneResource:                resourceDNSZone(),
			DNSZoneRecordResource:          resourceDNSZoneRecord(),
			"edgecenter_cdn_resource":      resourceCDNResource(),
			"edgecenter_cdn_origingroup":   resourceCDNOriginGroup(),
			"edgecenter_cdn_rule":          resourceCDNRule(),
			"edgecenter_cdn_sslcert":       resourceCDNCert(),
			LifecyclePolicyResource:        resourceLifecyclePolicy(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"edgecenter_project":           dataSourceProject(),
			"edgecenter_region":            dataSourceRegion(),
			"edgecenter_securitygroup":     dataSourceSecurityGroup(),
			"edgecenter_image":             dataSourceImage(),
			"edgecenter_volume":            dataSourceVolume(),
			"edgecenter_network":           dataSourceNetwork(),
			"edgecenter_subnet":            dataSourceSubnet(),
			"edgecenter_router":            dataSourceRouter(),
			"edgecenter_loadbalancer":      dataSourceLoadBalancer(),
			"edgecenter_loadbalancerv2":    dataSourceLoadBalancerV2(),
			"edgecenter_lblistener":        dataSourceLBListener(),
			"edgecenter_lbpool":            dataSourceLBPool(),
			"edgecenter_instance":          dataSourceInstance(),
			"edgecenter_floatingip":        dataSourceFloatingIP(),
			"edgecenter_storage_s3":        dataSourceStorageS3(),
			"edgecenter_storage_s3_bucket": dataSourceStorageS3Bucket(),
			"edgecenter_reservedfixedip":   dataSourceReservedFixedIP(),
			"edgecenter_servergroup":       dataSourceServerGroup(),
			"edgecenter_k8s":               dataSourceK8s(),
			"edgecenter_k8s_pool":          dataSourceK8sPool(),
			"edgecenter_k8s_client_config": dataSourceK8sClientConfig(),
			"edgecenter_secret":            dataSourceSecret(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			terraformVersion = "0.12+compatible"
		}
		return providerConfigure(ctx, d, terraformVersion)
	}

	return p
}

func providerConfigure(_ context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	username := d.Get("user_name").(string)
	password := d.Get("password").(string)
	permanentToken := d.Get(ProviderOptPermanentToken).(string)
	apiEndpoint := d.Get(ProviderOptSingleAPIEndpoint).(string)

	cloudAPI := d.Get("edgecenter_cloud_api").(string)
	if cloudAPI == "" {
		cloudAPI = d.Get("edgecenter_api").(string)
	}
	if cloudAPI == "" {
		cloudAPI = apiEndpoint + "/cloud"
	}

	cdnAPI := d.Get("edgecenter_cdn_api").(string)
	if cdnAPI == "" {
		cdnAPI = apiEndpoint
	}

	storageAPI := d.Get("edgecenter_storage_api").(string)
	if storageAPI == "" {
		storageAPI = apiEndpoint + "/storage"
	}

	dnsAPI := d.Get("edgecenter_dns_api").(string)
	if dnsAPI == "" {
		dnsAPI = apiEndpoint + "/dns"
	}

	platform := d.Get("edgecenter_platform_api").(string)
	if platform == "" {
		platform = d.Get("edgecenter_platform").(string)
	}
	if platform == "" {
		platform = apiEndpoint + "/iam"
	}

	clientID := d.Get("edgecenter_client_id").(string)

	var diags diag.Diagnostics

	var err error
	var provider *edgecloud.ProviderClient
	if permanentToken != "" {
		provider, err = ec.APITokenClient(edgecloud.APITokenOptions{
			APIURL:   cloudAPI,
			APIToken: permanentToken,
		})
	} else {
		provider, err = ec.AuthenticatedClient(edgecloud.AuthOptions{
			APIURL:      cloudAPI,
			AuthURL:     platform,
			Username:    username,
			Password:    password,
			AllowReauth: true,
			ClientID:    clientID,
		})
	}
	if err != nil {
		provider = &edgecloud.ProviderClient{}
		log.Printf("[WARN] init auth client: %s\n", err)
	}

	cdnProvider := eccdnProvider.NewClient(cdnAPI, eccdnProvider.WithSignerFunc(func(req *http.Request) error {
		for k, v := range provider.AuthenticatedHeaders() {
			req.Header.Set(k, v)
		}

		return nil
	}))
	cdnService := cdn.NewService(cdnProvider)

	config := Config{
		Provider:  provider,
		CDNClient: cdnService,
	}

	userAgent := fmt.Sprintf("terraform/%s", terraformVersion)
	if storageAPI != "" {
		stHost, stPath, err := ExtractHostAndPath(storageAPI)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf("storage api url: %w", err))
		}
		config.StorageClient = storageSDK.NewSDK(
			stHost,
			stPath,
			storageSDK.WithBearerAuth(provider.AccessToken),
			storageSDK.WithPermanentTokenAuth(func() string { return permanentToken }),
			storageSDK.WithUserAgent(userAgent),
		)
	}
	if dnsAPI != "" {
		baseURL, err := url.Parse(dnsAPI)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf("dns api url: %w", err))
		}
		authorizer := dnssdk.BearerAuth(provider.AccessToken())
		if permanentToken != "" {
			authorizer = dnssdk.PermanentAPIKeyAuth(permanentToken)
		}
		config.DNSClient = dnssdk.NewClient(
			authorizer,
			func(client *dnssdk.Client) {
				client.BaseURL = baseURL
				client.Debug = os.Getenv("TF_LOG") == "DEBUG"
			},
			func(client *dnssdk.Client) {
				client.UserAgent = userAgent
			})
	}

	return &config, diags
}
