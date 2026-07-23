package cdn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	cdnsdk "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/lecerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func resourceCDNLECert() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: importCDNLECert,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "ID ресурса CDN, к которому привязывается ACME сертификат (Let's Encrypt или Минцифры). Используется как ID при импорте. Нельзя изменить после создания.",
			},
			"cert_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(lecerts.CertTypeLE),
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{string(lecerts.CertTypeLE), string(lecerts.CertTypeMDDC)}, false),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Id() != "" && old == ""
				},
				Description: "Тип ACME сертификата. Допустимые значения: \"LE\" (Let's Encrypt) и \"MDDC\" (Минцифры/НУЦ Восход). По умолчанию \"LE\". Нельзя изменить после создания.",
			},
			"update": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Ручное обновление сертификата (перевыпуск). Работает как кнопка: после apply возвращается к false. Нельзя указывать true при создании.",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Отмена текущего процесса выпуска сертификата. false = отменить выпуск. Работает как кнопка: после apply возвращается к true. Нельзя указывать false при создании.",
			},
		},
		ReadContext:   resourceCDNLECertRead,
		CreateContext: resourceCDNLECertCreate,
		UpdateContext: resourceCDNLECertUpdate,
		DeleteContext: resourceCDNLECertDelete,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			if d.Id() == "" {
				if d.Get("update").(bool) {
					return fmt.Errorf("'update' cannot be set to true when creating the certificate")
				}
				if !d.Get("active").(bool) {
					return fmt.Errorf("'active' cannot be set to false when creating the certificate")
				}

				return nil
			}
			if d.HasChange("update") && d.HasChange("active") {
				return fmt.Errorf("you cannot change 'update' and 'active' at the same time")
			}
			if d.HasChange("cert_type") {
				if old, _ := d.GetChange("cert_type"); old.(string) != "" {
					return fmt.Errorf("cert_type cannot be changed; destroy the resource first to switch the certificate type")
				}
			}

			return nil
		},
	}
}

func importCDNLECert(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)

	resourceID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid import id %q, expected <resource_id> or <resource_id>:<cert_type>: %w", d.Id(), err)
	}
	if err := d.Set("resource_id", resourceID); err != nil {
		return nil, fmt.Errorf("set resource_id: %w", err)
	}

	if len(parts) == 2 {
		certType := parts[1]
		if certType != string(lecerts.CertTypeLE) && certType != string(lecerts.CertTypeMDDC) {
			return nil, fmt.Errorf("invalid cert_type %q in import id, expected %q or %q", certType, lecerts.CertTypeLE, lecerts.CertTypeMDDC)
		}
		if err := d.Set("cert_type", certType); err != nil {
			return nil, fmt.Errorf("set cert_type: %w", err)
		}
	}

	return []*schema.ResourceData{d}, nil
}

func resourceCDNLECertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*edgecenter.Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))
	certType := lecerts.CertType(d.Get("cert_type").(string))

	log.Printf("[DEBUG] Creating ACME cert (type=%s) for resource %d", certType, resourceID)
	var err error
	if certType == lecerts.CertTypeMDDC {
		err = client.LECerts().IssueLECert(ctx, resourceID, &lecerts.IssueRequest{CertType: certType})
	} else {
		err = client.LECerts().IssueLECert(ctx, resourceID, nil)
	}
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create ACME cert for resource %d: %w", resourceID, err))
	}
	log.Printf("[DEBUG] ACME cert creation finished for resource %d", resourceID)

	return resourceCDNLECertRead(ctx, d, m)
}

func resourceCDNLECertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*edgecenter.Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))

	log.Printf("[DEBUG] Reading ACME cert for resource %d", resourceID)
	cert, err := client.LECerts().GetLECert(ctx, resourceID)
	if err != nil {
		if errors.Is(err, cdnsdk.ErrNotFound) {
			d.SetId("")
			log.Printf("[DEBUG] ACME cert status not found for resource %d, clearing state", resourceID)

			return nil
		}

		return diag.FromErr(fmt.Errorf("failed to read ACME cert for resource %d: %w", resourceID, err))
	}

	if cert.ID == 0 && !cert.Active {
		d.SetId("")
		log.Printf("[DEBUG] ACME cert not found or inactive for resource %d, clearing state", resourceID)
		return nil
	}

	certType := cert.CertType
	if certType == "" {
		certType = lecerts.CertType(d.Get("cert_type").(string))
	}
	if certType == "" {
		certType = lecerts.CertTypeLE
	}

	d.SetId(fmt.Sprintf("%d", cert.ID))
	_ = d.Set("cert_type", string(certType))
	_ = d.Set("active", true)
	_ = d.Set("update", false)
	log.Printf("[DEBUG] Finished reading ACME cert: ID=%d, type=%s", cert.ID, certType)

	return nil
}

func resourceCDNLECertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*edgecenter.Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))
	flagUpdate := d.Get("update").(bool)
	active := d.Get("active").(bool)

	time.Sleep(1 * time.Second)
	cert, err := client.LECerts().GetLECert(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read ACME cert for resource %d: %w", resourceID, err))
	}
	r, err := client.Resources().Get(ctx, resourceID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read resource %d: %w", resourceID, err))
	}

	if flagUpdate && cert.ID == 0 {
		return diag.FromErr(fmt.Errorf("certificate for resource %d is not issued yet, nothing to renew", resourceID))
	}

	if flagUpdate && (cert.ID == r.SSLData) {
		log.Printf("[DEBUG] Updating ACME cert for resource %d", resourceID)
		if err = client.LECerts().UpdateLECert(ctx, resourceID); err != nil {
			return diag.FromErr(fmt.Errorf("failed to update ACME cert for resource %d: %w", resourceID, err))
		}
	}

	if !active {
		log.Printf("[DEBUG] Cancelling ACME cert for resource %d", resourceID)
		if err = client.LECerts().CancelLECert(ctx, resourceID, active); err != nil {
			return diag.FromErr(fmt.Errorf("failed to cancel ACME cert for resource %d: %w", resourceID, err))
		}
	}

	_ = d.Set("update", false)
	_ = d.Set("active", true)
	log.Printf("[DEBUG] Finished updating ACME cert for resource %d", resourceID)

	return nil
}

func resourceCDNLECertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*edgecenter.Config)
	client := config.CDNClient
	resourceID := int64(d.Get("resource_id").(int))

	log.Printf("[DEBUG] Deleting ACME cert for resource %d", resourceID)
	err := client.LECerts().DeleteLECert(ctx, resourceID, true)
	if err != nil && !errors.Is(err, cdnsdk.ErrBadRequest) && !errors.Is(err, cdnsdk.ErrNotFound) {
		return diag.FromErr(fmt.Errorf("failed to delete ACME cert for resource %d: %w", resourceID, err))
	}

	if errors.Is(err, cdnsdk.ErrBadRequest) {
		cert, gerr := client.LECerts().GetLECert(ctx, resourceID)
		if gerr != nil && !errors.Is(gerr, cdnsdk.ErrNotFound) {
			return diag.FromErr(fmt.Errorf("failed to delete ACME cert for resource %d: %w", resourceID, err))
		}
		if gerr == nil && cert.ID != 0 {
			return diag.FromErr(fmt.Errorf("failed to delete ACME cert for resource %d: %w", resourceID, err))
		}
		if gerr == nil && cert.ID == 0 && cert.Active {
			log.Printf("[DEBUG] Nothing issued to revoke for resource %d, cancelling pending issuance", resourceID)
			if cerr := client.LECerts().CancelLECert(ctx, resourceID, false); cerr != nil {
				return diag.FromErr(fmt.Errorf("failed to cancel ACME cert issuance for resource %d: %w", resourceID, cerr))
			}
		}
	}

	d.SetId("")
	log.Printf("[DEBUG] Finished deleting ACME cert for resource %d", resourceID)

	return nil
}
