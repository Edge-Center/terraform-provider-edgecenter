package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
)

func resourceProtectionResourceAliasCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceAliasCertificateCreateOrUpdate,
		ReadContext:   resourceProtectionResourceAliasCertificateRead,
		UpdateContext: resourceProtectionResourceAliasCertificateCreateOrUpdate,
		DeleteContext: resourceProtectionResourceAliasCertificateDelete,
		Description:   "Allows to manage certificates for aliases for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"alias": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of alias of DDoS protection resource. Has form `<resource_id>:<alias_id>`",
			},
			"ssl_crt": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Public part of the SSL certificate. It is required add all chains.",
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
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Let's Encrypt SSL certificate issuance status.",
			},
			"ssl_type": {
				Type:        schema.TypeString,
				Required:    true,
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
		},
	}
}

func resourceProtectionResourceAliasCertificateCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Get("alias").(string))
	log.Printf("[DEBUG] Setting certificate for alias for DDoS protection resource %s", d.Get("alias").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	aliasID, err := strconv.ParseInt(aID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.AliasUpdateRequest

	sslType := d.Get("ssl_type").(string)
	req.SSLType = &sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk("ssl_crt"); ok {
			sslcrtVal := sslcrt.(string)
			req.SSLCrt = &sslcrtVal
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

	_, _, err = client.Aliases.Update(ctx, resourceID, aliasID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, aliasID))
	resourceProtectionResourceAliasCertificateRead(ctx, d, m)

	log.Printf("[DEBUG] Finish setting certificate for alias for DDoS protection resource (id=%d:%d)\n", resourceID, aliasID)

	return nil
}

func resourceProtectionResourceAliasCertificateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start reading certificate for alias for DDoS protection resource (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	aliasID, err := strconv.ParseInt(aID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	result, _, err := client.Aliases.Get(ctx, resourceID, aliasID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("ssl_expire", result.SSLExpire)
	d.Set("ssl_status", result.SSLStatus)
	d.Set("ssl_type", result.SSLType)

	log.Println("[DEBUG] Finish reading alias for DDoS")

	return nil
}

func resourceProtectionResourceAliasCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start deleting certificate for alias for DDoS protection resource (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	aliasID, err := strconv.ParseInt(aID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	var req protectionSDK.AliasUpdateRequest

	if _, _, err := client.Aliases.Update(ctx, resourceID, aliasID, &req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish deleting certificate for DDoS protection resource alias")

	return nil
}
