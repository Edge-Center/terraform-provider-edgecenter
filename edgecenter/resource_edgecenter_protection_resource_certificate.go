package edgecenter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
)

const (
	sslEmpty  = ""
	sslCustom = "custom"
	sslLE     = "le"
)

func resourceProtectionResourceCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceCertificateCreateOrUpdate,
		ReadContext:   resourceProtectionResourceCertificateRead,
		UpdateContext: resourceProtectionResourceCertificateCreateOrUpdate,
		DeleteContext: resourceProtectionResourceCertificateDelete,
		Description:   "Allows to control SSL certificate for DDoS protection resource.",
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
				Description: "Public part of the SSL certificate. It is required add all chains.",
			},
			"ssl_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Private key of the SSL certificate.",
				Sensitive:   true,
			},
			"ssl_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("Select the SSL certificate type. Available values are `%s`, `%s`, `%s`.", sslEmpty, sslCustom, sslLE),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case sslEmpty, sslCustom, sslLE:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`.", v, sslEmpty, sslCustom, sslLE)
				},
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
	req.SSLType = sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk("ssl_crt"); ok {
			req.SSLCert = sslcrt.(string)
		} else {
			return diag.Errorf("No certificate set for %s", resourceID)
		}

		if sslkey, ok := d.GetOk("ssl_key"); ok {
			req.SSLKey = sslkey.(string)
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

	d.Set("resource", result.ID)
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

	req.SSLType = ""
	req.SSLCert = ""
	req.SSLKey = ""

	_, _, err = client.Resources.Update(ctx, id, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish deleting DDoS protection resource certificate")

	return nil
}
