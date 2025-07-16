package edgecenter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
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
			"resource": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of DDoS protection resource to manage certificate for.",
			},
			"ssl_crt": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Public part of the SSL certificate. It is required add all chains. Each certificate chain should be separated by `\\n`.",
			},
			"ssl_expire": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "UNIX timestamp of the SSL certificate expiration date.",
			},
			"ssl_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Private key of the SSL certificate.",
				Sensitive:   true,
			},
			"ssl_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Let's Encrypt SSL certificate issuance status.",
			},
			"ssl_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  fmt.Sprintf("Select the SSL certificate type. Available values are `%s`, `%s`.", sslCustom, sslLE),
				ValidateFunc: validation.StringInSlice([]string{sslCustom, sslLE}, false),
			},
		},
	}
}

func resourceProtectionResourceCertificateCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get("resource").(string)
	log.Printf("[DEBUG] Setting certificate for DDoS protection resource %s", resourceID)
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

	jreq, _ := json.Marshal(result)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.ResourceUpdateRequest

	json.Unmarshal(jreq, &req)

	sslType := d.Get("ssl_type").(string)
	req.SSLType = &sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk("ssl_crt"); ok {
			sslcrtVal := sslcrt.(string)
			req.SSLCert = &sslcrtVal
		} else {
			return diag.Errorf("No certificate set for %s", resourceID)
		}

		if sslkey, ok := d.GetOk("ssl_key"); ok {
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

	log.Printf("[DEBUG] Finish setting certificate for DDoS protection resource (id=%d)\n", resourceID)

	return nil
}

func resourceProtectionResourceCertificateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start reading DDoS protection resource certificate type (id=%s)\n", resourceID)
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

	d.Set("resource", resourceID)
	d.Set("ssl_expire", result.SSLExpire)
	d.Set("ssl_status", result.SSLStatus)
	d.Set("ssl_type", result.SSLType)

	log.Println("[DEBUG] Finish reading DDoS protection resource certificate type")

	return nil
}

func resourceProtectionResourceCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start deleting DDoS protection resource certificate (id=%s)\n", resourceID)
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

	jreq, _ := json.Marshal(result)
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
