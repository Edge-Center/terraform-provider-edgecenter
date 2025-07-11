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

func resourceProtectionResourceAlias() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceAliasCreate,
		ReadContext:   resourceProtectionResourceAliasRead,
		UpdateContext: resourceProtectionResourceAliasUpdate,
		DeleteContext: resourceProtectionResourceAliasDelete,
		Description:   "Allows to manage aliases for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of alias of DDoS protection resource. Must be a sub-domain of resource..",
			},
			"resource": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of DDoS protection resource to manage alias for.",
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
				Optional:    true,
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
		},
	}
}

func resourceProtectionResourceAliasCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get("resource").(string)
	log.Printf("[DEBUG] Start creating alias for DDoS protection resource %s", resourceID)
	config := m.(*Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.AliasCreateRequest

	req.Name = d.Get("name").(string)

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

	result, _, err := client.Aliases.Create(ctx, id, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceAliasRead(ctx, d, m)

	log.Printf("[DEBUG] Finish creating alias for DDoS protection resource (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceAliasRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start reading alias for DDoS protection resource (id=%s)\n", d.Id())
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

	d.Set("resource", fmt.Sprintf("%d", resourceID))
	d.Set("ssl_type", result.SSLType)

	log.Println("[DEBUG] Finish reading alias for DDoS")

	return nil
}

func resourceProtectionResourceAliasUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start updating alias for DDoS protection resource (id=%s)\n", d.Id())
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

	sslType := d.Get("ssl_type").(string)
	req.SSLType = &sslType

	if sslType == sslCustom {
		if sslcrt, ok := d.GetOk("ssl_crt"); ok {
			sslcrtVal := sslcrt.(string)
			req.SSLCrt = &sslcrtVal
		} else {
			return diag.Errorf("No certificate set for %d:%d", resourceID, aliasID)
		}

		if sslkey, ok := d.GetOk("ssl_key"); ok {
			sslkeyVal := sslkey.(string)
			req.SSLKey = &sslkeyVal
		} else {
			return diag.Errorf("No certificate key set for %d:%d", resourceID, aliasID)
		}
	}

	_, _, err = client.Aliases.Update(ctx, resourceID, aliasID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish updating alias for DDoS protection resource")

	return resourceProtectionResourceAliasRead(ctx, d, m)
}

func resourceProtectionResourceAliasDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start updating alias for DDoS protection resource (id=%s)\n", d.Id())
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

	if _, err := client.Blacklists.Delete(ctx, resourceID, aliasID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish Protection Resource Blacklist entry deleting")

	return nil
}
