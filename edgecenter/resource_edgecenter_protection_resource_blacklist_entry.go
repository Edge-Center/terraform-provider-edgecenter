package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
)

func resourceProtectionResourceBlacklistEntry() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceBlacklistEntryCreate,
		ReadContext:   resourceProtectionResourceBlacklistEntryRead,
		UpdateContext: resourceProtectionResourceBlacklistEntryUpdate,
		DeleteContext: resourceProtectionResourceBlacklistEntryDelete,
		Description:   "Represent IP added to blacklist for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Blocked IP address.",
			},
			"resource": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the DDoS protection resource to add header to.",
			},
		},
	}
}

func resourceProtectionResourceBlacklistEntryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Protection Resource Blacklist entry creating")
	config := m.(*Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(d.Get("resource").(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.BlacklistCreateRequest

	req.IP = d.Get("ip").(string)

	result, _, err := client.Blacklists.Create(ctx, resourceID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceBlacklistEntryRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Protection Resource Blacklist entry creating (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceBlacklistEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Blacklist entry reading (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	entryID, err := strconv.ParseInt(eID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	result, _, err := client.Blacklists.Get(ctx, resourceID, entryID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("resource", fmt.Sprintf("%d", resourceID))
	d.Set("ip", result.IP)

	log.Println("[DEBUG] Finish Protection Resource Blacklist entry reading")

	return nil
}

func resourceProtectionResourceBlacklistEntryUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Blacklist entry updating (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	entryID, err := strconv.ParseInt(eID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	var req protectionSDK.BlacklistCreateRequest

	req.IP = d.Get("ip").(string)

	if _, _, err := client.Blacklists.Update(ctx, resourceID, entryID, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Protection Resource Blacklist entry updating")

	return resourceProtectionResourceBlacklistEntryRead(ctx, d, m)
}

func resourceProtectionResourceBlacklistEntryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Blacklist entry deleting (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	entryID, err := strconv.ParseInt(eID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	if _, err := client.Blacklists.Delete(ctx, resourceID, entryID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish Protection Resource Blacklist entry deleting")

	return nil
}
