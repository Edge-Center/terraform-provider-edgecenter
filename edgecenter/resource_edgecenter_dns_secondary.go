package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	dnssdk "github.com/Edge-Center/edgecenter-dns-sdk-go"
)

const (
	DNSSecondaryZoneResource        = "edgecenter_dns_secondary_zone"
	DNSSecondaryZoneSchemaName      = "name"
	DNSSecondaryZoneSchemaMaster    = "master"
	DNSSecondaryZoneSchemaTSIGKey   = "tsig_key"
	DNSSecondaryZoneSchemaTSIGName  = "tsig_name"
	DNSSecondaryZoneSchemaZoneID    = "zone_id"
	DNSSecondaryZoneSchemaUpdatedAt = "updated_at"
)

func resourceDNSSecondaryZone() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			DNSSecondaryZoneSchemaName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					zoneName := i.(string)
					if strings.TrimSpace(zoneName) == "" || len(zoneName) > 255 {
						return diag.Errorf("secondary zone name can't be empty, it also should be less than 256 symbols")
					}
					// validate zoneName
					if !strings.Contains(zoneName, ".") {
						return diag.Errorf("secondary zone name must be a valid domain name (e.g., example.com)")
					}
					return nil
				},
				Description: "A name of DNS Secondary Zone resource.",
			},
			DNSSecondaryZoneSchemaMaster: {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					master := i.(string)
					if strings.TrimSpace(master) == "" {
						return diag.Errorf("master server address can't be empty")
					}
					// validate IP or domain name
					if !strings.Contains(master, ".") {
						return diag.Errorf("master server must be a valid IP address or domain name")
					}
					return nil
				},
				Description: "IP address of the primary DNS server for the secondary zone.",
			},
			DNSSecondaryZoneSchemaTSIGKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Base64 encoded TSIG key value for secure zone transfer.",
			},
			DNSSecondaryZoneSchemaTSIGName: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "TSIG key name in the format: keyName.zoneName",
			},
			DNSSecondaryZoneSchemaZoneID: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Unique identifier of the secondary zone.",
			},
			DNSSecondaryZoneSchemaUpdatedAt: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last update of the secondary zone.",
			},
		},
		CreateContext: checkDNSDependency(resourceDNSSecondaryZoneCreate),
		ReadContext:   checkDNSDependency(resourceDNSSecondaryZoneRead),
		UpdateContext: checkDNSDependency(resourceDNSSecondaryZoneUpdate),
		DeleteContext: checkDNSDependency(resourceDNSSecondaryZoneDelete),
		Description:   "Represent DNS Secondary Zone resource. Secondary zones allow you to host a read-only copy of a zone from another DNS server.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceDNSSecondaryZoneCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := strings.TrimSpace(d.Get(DNSSecondaryZoneSchemaName).(string))
	master := strings.TrimSpace(d.Get(DNSSecondaryZoneSchemaMaster).(string))

	log.Printf("[DEBUG] Start DNS Secondary Zone Resource creating: name=%s, master=%s\n", name, master)
	defer log.Printf("[DEBUG] Finish DNS Secondary Zone Resource creating (id=%s)\n", name)

	config := m.(*Config)
	client := config.DNSClient

	existingZone, err := client.GetSecondaryZone(ctx, name)
	if err == nil {
		log.Printf("[INFO] Secondary zone %s already exists, importing to state", name)

		d.SetId(name)
		return setSecondaryZoneData(d, existingZone)
	}

	// 404 error is valid
	var apiErr dnssdk.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] Zone %s not found, will create new one", name)
	} else {
		return diag.FromErr(fmt.Errorf("check existing secondary zone: %w", err))
	}

	// create new secondary zone
	createReq := dnssdk.CreateSecondaryZoneRequest{
		Name:   name,
		Master: master,
	}

	if tsigKey, ok := d.GetOk(DNSSecondaryZoneSchemaTSIGKey); ok {
		createReq.Key = tsigKey.(string)
	}
	if tsigName, ok := d.GetOk(DNSSecondaryZoneSchemaTSIGName); ok {
		createReq.TSIGName = tsigName.(string)
	}

	zone, err := client.CreateSecondaryZone(ctx, createReq)
	if err != nil {
		return diag.FromErr(fmt.Errorf("create secondary zone: %w", err))
	}

	d.SetId(name)

	return setSecondaryZoneData(d, zone)
}

func resourceDNSSecondaryZoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	zoneName := dnsSecondaryZoneResourceID(d)
	log.Printf("[DEBUG] Start DNS Secondary Zone Resource reading (id=%s)\n", zoneName)
	defer log.Println("[DEBUG] Finish DNS Secondary Zone Resource reading")

	if zoneName == "" {
		return diag.Errorf("empty secondary zone name")
	}

	config := m.(*Config)
	client := config.DNSClient

	zone, err := client.GetSecondaryZone(ctx, zoneName)
	if err != nil {
		log.Printf("[DEBUG] GetSecondaryZone error: %v (type: %T)\n", err, err)

		// is zone deleted
		var apiErr dnssdk.APIError
		if errors.As(err, &apiErr) {
			log.Printf("[DEBUG] API Error - StatusCode: %d, Message: %s\n", apiErr.StatusCode, apiErr.Message)
			if apiErr.StatusCode == http.StatusNotFound {
				log.Printf("[WARN] Secondary zone %s not found, removing from state", zoneName)
				d.SetId("")
				return nil
			}
		}

		return diag.FromErr(fmt.Errorf("get secondary zone: %w", err))
	}

	log.Printf("[DEBUG] Successfully found zone: %+v\n", zone)

	return setSecondaryZoneData(d, zone)
}

func resourceDNSSecondaryZoneUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	zoneName := dnsSecondaryZoneResourceID(d)
	log.Printf("[DEBUG] Start DNS Secondary Zone Resource updating (id=%s)\n", zoneName)
	defer log.Println("[DEBUG] Finish DNS Secondary Zone Resource updating")

	if zoneName == "" {
		return diag.Errorf("empty secondary zone name")
	}

	config := m.(*Config)
	client := config.DNSClient

	updateReq := dnssdk.UpdateSecondaryZoneRequest{
		Master: d.Get(DNSSecondaryZoneSchemaMaster).(string),
	}

	if tsigKey, ok := d.GetOk(DNSSecondaryZoneSchemaTSIGKey); ok {
		updateReq.Key = tsigKey.(string)
	}

	if tsigName, ok := d.GetOk(DNSSecondaryZoneSchemaTSIGName); ok {
		updateReq.Name = tsigName.(string)
	}

	// send PUT request
	zone, err := client.UpdateSecondaryZone(ctx, zoneName, updateReq)
	if err != nil {
		return diag.FromErr(fmt.Errorf("update secondary zone: %w", err))
	}

	return setSecondaryZoneData(d, zone)
}

func resourceDNSSecondaryZoneDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	zoneName := dnsSecondaryZoneResourceID(d)
	log.Printf("[DEBUG] Start DNS Secondary Zone Resource deleting (id=%s)\n", zoneName)
	defer log.Println("[DEBUG] Finish DNS Secondary Zone Resource deleting")

	if zoneName == "" {
		return diag.Errorf("empty secondary zone name")
	}

	config := m.(*Config)
	client := config.DNSClient

	// delete secondary zone
	err := client.DeleteSecondaryZone(ctx, zoneName)
	if err != nil {
		// check for already deleted
		var apiErr dnssdk.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Secondary zone %s already deleted", zoneName)
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("delete secondary zone: %w", err))
	}

	d.SetId("")

	return nil
}

func dnsSecondaryZoneResourceID(d *schema.ResourceData) string {
	resourceID := d.Id()
	if resourceID == "" {
		resourceID = d.Get(DNSSecondaryZoneSchemaName).(string)
	}
	return strings.TrimSpace(resourceID)
}

func setSecondaryZoneData(d *schema.ResourceData, zone dnssdk.SecondaryZone) diag.Diagnostics {
	d.SetId(zone.Name)

	if err := d.Set(DNSSecondaryZoneSchemaName, zone.Name); err != nil {
		return diag.FromErr(fmt.Errorf("set name: %w", err))
	}

	if err := d.Set(DNSSecondaryZoneSchemaZoneID, zone.ID); err != nil {
		return diag.FromErr(fmt.Errorf("set zone_id: %w", err))
	}

	var updatedAtStr string
	if zone.UpdatedAt != 0 {
		// convert nanoseconds to RFC3339 string
		t := time.Unix(0, int64(zone.UpdatedAt))
		updatedAtStr = t.Format(time.RFC3339)
	}

	if err := d.Set(DNSSecondaryZoneSchemaUpdatedAt, updatedAtStr); err != nil {
		return diag.FromErr(fmt.Errorf("set updated_at: %w", err))
	}

	if zone.TSIG != nil && zone.TSIG.Master != "" {
		if err := d.Set(DNSSecondaryZoneSchemaMaster, zone.TSIG.Master); err != nil {
			return diag.FromErr(fmt.Errorf("set master: %w", err))
		}

		if zone.TSIG.Name != "" {
			if err := d.Set(DNSSecondaryZoneSchemaTSIGName, zone.TSIG.Name); err != nil {
				return diag.FromErr(fmt.Errorf("set tsig_name: %w", err))
			}
		}
	}

	return nil
}

// dataSourceDNSSecondaryZonesRead reads all secondary zones.
func dataSourceDNSSecondaryZonesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DNS Secondary Zones Data Source reading")
	defer log.Println("[DEBUG] Finish DNS Secondary Zones Data Source reading")

	config := m.(*Config)
	client := config.DNSClient

	zones, err := client.SecondaryZones(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("get secondary zones: %w", err))
	}

	// convert to Terraform format
	zoneList := make([]map[string]interface{}, len(zones))
	for i, zone := range zones {
		// convert Timestamp → string
		var updatedAtStr string
		if zone.UpdatedAt != 0 {
			t := time.Unix(0, int64(zone.UpdatedAt))
			updatedAtStr = t.Format(time.RFC3339)
		}

		zoneMap := map[string]interface{}{
			DNSSecondaryZoneSchemaName:      zone.Name,
			DNSSecondaryZoneSchemaZoneID:    zone.ID,
			DNSSecondaryZoneSchemaUpdatedAt: updatedAtStr, // ← строка
		}

		if zone.TSIG != nil {
			zoneMap[DNSSecondaryZoneSchemaMaster] = zone.TSIG.Master
			zoneMap[DNSSecondaryZoneSchemaTSIGName] = zone.TSIG.Name
			// TSIG Key is hidden fir security reasons
		}

		zoneList[i] = zoneMap
	}

	if err := d.Set("zones", zoneList); err != nil {
		return diag.FromErr(fmt.Errorf("set zones: %w", err))
	}

	d.SetId("secondary_zones")

	return nil
}
