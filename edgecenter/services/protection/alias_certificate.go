package protection

import (
	"context"
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
	ProtectionAliasCertificateResource = "edgecenter_protection_resource_alias_certificate"

	ProtectionAliasCertificateSchemaAlias     = "alias"
	ProtectionAliasCertificateSchemaSSLCrt    = "ssl_crt"
	ProtectionAliasCertificateSchemaSSLExpire = "ssl_expire"
	ProtectionAliasCertificateSchemaSSLKey    = "ssl_key"
	ProtectionAliasCertificateSchemaSSLStatus = "ssl_status"
	ProtectionAliasCertificateSchemaSSLType   = "ssl_type"
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
			ProtectionAliasCertificateSchemaAlias: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of alias of DDoS protection resource. Has form `<resource_id>:<alias_id>`",
			},
			ProtectionAliasCertificateSchemaSSLCrt: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Public part of the SSL certificate. Add all the certificate chains. Each certificate chain should be separated by `\\n`",
			},
			ProtectionAliasCertificateSchemaSSLExpire: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "UNIX timestamp of the SSL certificate expiration date.",
			},
			ProtectionAliasCertificateSchemaSSLKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Private key of the SSL certificate.",
				Sensitive:   true,
			},
			ProtectionAliasCertificateSchemaSSLStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Let's Encrypt SSL certificate issuance status.",
			},
			ProtectionAliasCertificateSchemaSSLType: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  fmt.Sprintf("Select the SSL certificate type. Available values are `%s`, `%s`.", sslCustom, sslLE),
				ValidateFunc: validation.StringInSlice([]string{sslCustom, sslLE}, false),
			},
		},
	}
}

func resourceProtectionResourceAliasCertificateCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := edgecenter.ImportStringParserSimple(d.Get(ProtectionAliasCertificateSchemaAlias).(string))
	log.Printf("[DEBUG] Setting certificate for alias for DDoS protection resource %s", d.Get(ProtectionAliasCertificateSchemaAlias).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*edgecenter.Config)
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

	sslType := d.Get(ProtectionAliasCertificateSchemaSSLType).(string)
	req.SSLType = &sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk(ProtectionAliasCertificateSchemaSSLCrt); ok {
			sslcrtVal := sslcrt.(string)
			req.SSLCrt = &sslcrtVal
		} else {
			return diag.Errorf("No certificate set for %d", resourceID)
		}

		if sslkey, ok := d.GetOk(ProtectionAliasCertificateSchemaSSLKey); ok {
			sslkeyVal := sslkey.(string)
			req.SSLKey = &sslkeyVal
		} else {
			return diag.Errorf("No certificate key set for %d", resourceID)
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
	rID, aID, err := edgecenter.ImportStringParserSimple(d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	result, _, err := client.Aliases.Get(ctx, resourceID, aliasID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(ProtectionAliasCertificateSchemaAlias, fmt.Sprintf("%d:%d", resourceID, aliasID))
	d.Set(ProtectionAliasCertificateSchemaSSLExpire, result.SSLExpire)
	d.Set(ProtectionAliasCertificateSchemaSSLStatus, result.SSLStatus)
	d.Set(ProtectionAliasCertificateSchemaSSLType, result.SSLType)

	log.Println("[DEBUG] Finish reading certificate for alias for DDoS")

	return nil
}

func resourceProtectionResourceAliasCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := edgecenter.ImportStringParserSimple(d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	var req protectionSDK.AliasUpdateRequest

	if _, _, err := client.Aliases.Update(ctx, resourceID, aliasID, &req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish deleting certificate for DDoS protection resource alias")

	return nil
}
