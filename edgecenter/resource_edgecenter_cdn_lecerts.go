package edgecenter

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCDNLECert() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "ID ресурса CDN, к которому привязывается Let's Encrypt сертификат",
			},
			"update": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Флаг обновления Let's Encrypt сертификата",
			},
		},
		ReadContext:   resourceCDNLECertRead,
		CreateContext: resourceCDNLECertCreate,
		UpdateContext: resourceCDNLECertUpdate,
		DeleteContext: resourceCDNLECertDelete,
	}
}

func resourceCDNLECertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LE cert create")
	config := m.(*Config)
	client := config.CDNClient

	resourceID := int64(d.Get("resource_id").(int))

	err := client.LECerts().CreateLECert(ctx, resourceID)
	if err != nil {
		log.Printf("[ERROR] Failed to create LE cert for resource ID %d: %v", resourceID, err)
	}
	resourceCDNLECertRead(ctx, d, m)

	log.Println("[DEBUG] Finished create LE cert")

	return nil
}

func resourceCDNLECertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	certID := d.Id()
	log.Printf("[DEBUG] Start read LE cert (%s)", certID)

	config := m.(*Config)
	client := config.CDNClient

	resourceID := int64(d.Get("resource_id").(int))

	req, err := client.LECerts().GetLECert(ctx, resourceID)
	if err != nil {
		log.Printf("[ERROR] Failed to read LE cert for resource ID %d: %v", resourceID, err)
	}

	d.SetId(fmt.Sprintf("%d", req.ID))

	log.Println("[DEBUG] Finished read LE cert")

	return nil
}

func resourceCDNLECertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	client := config.CDNClient

	flagUpdate := d.Get("update").(bool)

	if flagUpdate {
		log.Println("[DEBUG] Start update LE cert")

		resourceID := int64(d.Get("resource_id").(int))

		if err := client.LECerts().UpdateLECert(ctx, resourceID); err != nil {
			log.Printf("[ERROR] Failed to update LE cert for resource ID %d: %v", resourceID, err)
		}
		log.Println("[DEBUG] Finished update LE cert")
	}

	resourceCDNLECertRead(ctx, d, m)

	return nil
}

func resourceCDNLECertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	certID := d.Id()
	log.Printf("[DEBUG] Start delete LE cert (id=%s)\n", certID)
	config := m.(*Config)
	client := config.CDNClient

	resourceID := int64(d.Get("resource_id").(int))

	if err := client.LECerts().DeleteLECert(ctx, resourceID, false); err != nil {
		return diag.FromErr(fmt.Errorf("[ERROR] Failed to deleting LE cert for resource ID %d: %w", resourceID, err))
	}

	d.SetId("")
	log.Println("[DEBUG] Finished delete LE cert")

	return nil
}
