package edgecenter

import (
	"context"
	"fmt"
	"log"
	"time"

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
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Флаг отмены активного процесса выдачи SSL-сертификата Let's Encrypt.",
			},
		},
		ReadContext:   resourceCDNLECertRead,
		CreateContext: resourceCDNLECertCreate,
		UpdateContext: resourceCDNLECertUpdate,
		DeleteContext: resourceCDNLECertDelete,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			if d.HasChange("update") && d.HasChange("active") {
				return fmt.Errorf("you cannot change 'update' and 'active' at the same time")
			}
			return nil
		},
	}
}

func resourceCDNLECertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))

	log.Println("[DEBUG] Creating LE cert...")
	if err := client.LECerts().CreateLECert(ctx, resourceID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to create LE cert for resource %d: %w", resourceID, err))
	}
	log.Println("[DEBUG] LE cert creation finished.")

	return resourceCDNLECertRead(ctx, d, m)
}

func resourceCDNLECertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))

	log.Printf("[DEBUG] Reading LE cert for resource %d", resourceID)
	cert, err := client.LECerts().GetLECert(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read LE cert for resource %d: %w", resourceID, err))
	}

	if cert.ID == 0 && !cert.Active {
		d.SetId("")
		log.Printf("[DEBUG] LE cert not found or inactive for resource %d, clearing state", resourceID)
		return nil
	}

	d.SetId(fmt.Sprintf("%d", cert.ID))
	_ = d.Set("active", true)
	_ = d.Set("update", false)
	log.Printf("[DEBUG] Finished reading LE cert: ID=%d", cert.ID)

	return nil
}

func resourceCDNLECertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))
	flagUpdate := d.Get("update").(bool)
	active := d.Get("active").(bool)

	time.Sleep(1 * time.Second)
	cert, err := client.LECerts().GetLECert(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read LE cert for resource %d: %w", resourceID, err))
	}
	r, err := client.Resources().Get(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read resource %d: %w", resourceID, err))
	}

	if flagUpdate && (cert.ID == r.SSLData) {
		log.Printf("[DEBUG] Updating LE cert for resource %d", resourceID)
		if err = client.LECerts().UpdateLECert(ctx, resourceID); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update LE cert for resource %d: %w", resourceID, err))
		}
	}

	if !active {
		log.Printf("[DEBUG] Cancelling LE cert for resource %d", resourceID)
		if err = client.LECerts().CancelLECert(ctx, resourceID, active); err != nil {
			return diag.FromErr(fmt.Errorf("failed to cancel LE cert for resource %d: %w", resourceID, err))
		}
	}

	_ = d.Set("update", false)
	_ = d.Set("active", true)
	log.Printf("[DEBUG] Finished updating LE cert for resource %d", resourceID)

	return nil
}

func resourceCDNLECertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))

	log.Printf("[DEBUG] Deleting LE cert for resource %d", resourceID)
	if err := client.LECerts().DeleteLECert(ctx, resourceID, true); err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete LE cert for resource %d: %w", resourceID, err))
	}

	d.SetId("")
	log.Printf("[DEBUG] Finished deleting LE cert for resource %d", resourceID)

	return nil
}
