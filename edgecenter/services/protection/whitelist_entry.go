package protection

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	ProtectionWhitelistEntryResource = "edgecenter_protection_resource_whitelist_entry"

	ProtectionWhitelistEntrySchemaIP       = "ip"
	ProtectionWhitelistEntrySchemaResource = "resource"
)

func resourceProtectionResourceWhitelistEntry() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceWhitelistEntryCreate,
		ReadContext:   resourceProtectionResourceWhitelistEntryRead,
		UpdateContext: resourceProtectionResourceWhitelistEntryUpdate,
		DeleteContext: resourceProtectionResourceWhitelistEntryDelete,
		Description:   "Represent IP added to whitelist for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ProtectionWhitelistEntrySchemaIP: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Allowed IP address.",
			},
			ProtectionWhitelistEntrySchemaResource: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the DDoS protection resource to add header to.",
			},
		},
	}
}

func resourceProtectionResourceWhitelistEntryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DDoS Protection Resource Whitelist entry creating")
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(d.Get(ProtectionWhitelistEntrySchemaResource).(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.WhitelistCreateRequest

	req.IP = d.Get(ProtectionWhitelistEntrySchemaIP).(string)

	result, _, err := client.Whitelists.Create(ctx, resourceID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceWhitelistEntryRead(ctx, d, m)

	log.Printf("[DEBUG] Finish DDoS Protection Resource Whitelist entry creating (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceWhitelistEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Whitelist entry reading (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	result, _, err := client.Whitelists.Get(ctx, resourceID, entryID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(ProtectionWhitelistEntrySchemaResource, fmt.Sprintf("%d", resourceID))
	d.Set(ProtectionWhitelistEntrySchemaIP, result.IP)

	log.Println("[DEBUG] Finish DDoS Protection Resource Whitelist entry reading")

	return nil
}

func resourceProtectionResourceWhitelistEntryUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Whitelist entry updating (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	var req protectionSDK.WhitelistCreateRequest

	req.IP = d.Get(ProtectionWhitelistEntrySchemaIP).(string)

	if _, _, err := client.Whitelists.Update(ctx, resourceID, entryID, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish DDoS Protection Resource Whitelist entry updating")

	return resourceProtectionResourceWhitelistEntryRead(ctx, d, m)
}

func resourceProtectionResourceWhitelistEntryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, eID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Whitelist entry deleting (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	if _, err := client.Whitelists.Delete(ctx, resourceID, entryID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish DDoS Protection Resource Whitelist entry deleting")

	return nil
}
