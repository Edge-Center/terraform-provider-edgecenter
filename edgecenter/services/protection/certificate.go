package protection

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	ProtectionCertificateResource = "edgecenter_protection_resource_certificate"

	ProtectionCertificateSchemaResource  = "resource"
	ProtectionCertificateSchemaSSLCrt    = "ssl_crt"
	ProtectionCertificateSchemaSSLExpire = "ssl_expire"
	ProtectionCertificateSchemaSSLKey    = "ssl_key"
	ProtectionCertificateSchemaSSLStatus = "ssl_status"
	ProtectionCertificateSchemaSSLType   = "ssl_type"
)

func resourceProtectionResourceCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceCertificateCreateOrUpdate,
		ReadContext:   resourceProtectionResourceCertificateRead,
		UpdateContext: resourceProtectionResourceCertificateCreateOrUpdate,
		DeleteContext: resourceProtectionResourceCertificateDelete,
		Description:   "Allows to manage SSL certificate for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ProtectionCertificateSchemaResource: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of DDoS protection resource to manage certificate for.",
			},
			ProtectionCertificateSchemaSSLCrt: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Public part of the SSL certificate. It is required add all chains. Each certificate chain should be separated by `\\n`.",
			},
			ProtectionCertificateSchemaSSLExpire: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "UNIX timestamp of the SSL certificate expiration date.",
			},
			ProtectionCertificateSchemaSSLKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Private key of the SSL certificate.",
				Sensitive:   true,
			},
			ProtectionCertificateSchemaSSLStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Let's Encrypt SSL certificate issuance status.",
			},
			ProtectionCertificateSchemaSSLType: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  fmt.Sprintf("Select the SSL certificate type. Available values are `%s`, `%s`.", sslCustom, sslLE),
				ValidateFunc: validation.StringInSlice([]string{sslCustom, sslLE}, false),
			},
		},
	}
}

func resourceProtectionResourceCertificateCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get(ProtectionCertificateSchemaResource).(string)
	log.Printf("[DEBUG] Setting certificate for DDoS protection resource %s", resourceID)
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

	jreq, err := json.Marshal(result)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.ResourceUpdateRequest

	json.Unmarshal(jreq, &req)

	sslType := d.Get(ProtectionCertificateSchemaSSLType).(string)
	req.SSLType = &sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk(ProtectionCertificateSchemaSSLCrt); ok {
			sslcrtVal := sslcrt.(string)
			req.SSLCert = &sslcrtVal
		} else {
			return diag.Errorf("No certificate set for %s", resourceID)
		}

		if sslkey, ok := d.GetOk(ProtectionCertificateSchemaSSLKey); ok {
			sslkeyVal := sslkey.(string)
			req.SSLKey = &sslkeyVal
		} else {
			return diag.Errorf("No certificate key set for %s", resourceID)
		}
	}

	_, _, err = client.Resources.Update(ctx, id, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceID)
	resourceProtectionResourceCertificateRead(ctx, d, m)

	log.Printf("[DEBUG] Finish setting certificate for DDoS protection resource (id=%s)\n", resourceID)

	return nil
}

func resourceProtectionResourceCertificateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start reading DDoS protection resource certificate type (id=%s)\n", resourceID)
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

	d.Set(ProtectionCertificateSchemaResource, resourceID)
	d.Set(ProtectionCertificateSchemaSSLExpire, result.SSLExpire)
	d.Set(ProtectionCertificateSchemaSSLStatus, result.SSLStatus)
	d.Set(ProtectionCertificateSchemaSSLType, result.SSLType)

	log.Println("[DEBUG] Finish reading DDoS protection resource certificate type")

	return nil
}

func resourceProtectionResourceCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start deleting DDoS protection resource certificate (id=%s)\n", resourceID)
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

	jreq, err := json.Marshal(result)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.ResourceUpdateRequest

	json.Unmarshal(jreq, &req)

	req.SSLType = nil
	req.SSLCert = nil
	req.SSLKey = nil

	_, _, err = client.Resources.Update(ctx, id, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish deleting DDoS protection resource certificate")

	return nil
}
