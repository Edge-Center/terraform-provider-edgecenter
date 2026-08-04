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
	ProtectionHeaderResource = "edgecenter_protection_resource_header"

	ProtectionHeaderSchemaKey      = "key"
	ProtectionHeaderSchemaResource = "resource"
	ProtectionHeaderSchemaValue    = "value"
)

func resourceProtectionResourceHeader() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceHeaderCreate,
		ReadContext:   resourceProtectionResourceHeaderRead,
		UpdateContext: resourceProtectionResourceHeaderUpdate,
		DeleteContext: resourceProtectionResourceHeaderDelete,
		Description:   "Represent additional HTTP header returned to user by DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ProtectionHeaderSchemaKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "HTTP header name.",
			},
			ProtectionHeaderSchemaResource: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the DDoS protection resource to add header to.",
			},
			ProtectionHeaderSchemaValue: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "HTTP header value.",
			},
		},
	}
}

func resourceProtectionResourceHeaderCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DDoS Protection Resource Header creating")
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(d.Get(ProtectionHeaderSchemaResource).(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.HeaderCreateRequest

	req.Key = d.Get(ProtectionHeaderSchemaKey).(string)
	req.Value = d.Get(ProtectionHeaderSchemaValue).(string)

	result, _, err := client.Headers.Create(ctx, resourceID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceHeaderRead(ctx, d, m)

	log.Printf("[DEBUG] Finish DDoS Protection Resource Header creating (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceHeaderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, hID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Header reading (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	headerID, err := strconv.ParseInt(hID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	result, _, err := client.Headers.Get(ctx, resourceID, headerID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(ProtectionHeaderSchemaResource, fmt.Sprintf("%d", resourceID))
	d.Set(ProtectionHeaderSchemaKey, result.Key)
	d.Set(ProtectionHeaderSchemaValue, result.Value)

	log.Println("[DEBUG] Finish DDoS Protection Resource Header reading")

	return nil
}

func resourceProtectionResourceHeaderUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, hID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Header updating (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	headerID, err := strconv.ParseInt(hID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	var req protectionSDK.HeaderCreateRequest

	req.Key = d.Get(ProtectionHeaderSchemaKey).(string)
	req.Value = d.Get(ProtectionHeaderSchemaValue).(string)

	if _, _, err := client.Headers.Update(ctx, resourceID, headerID, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish DDoS Protection Resource Header updating")

	return resourceProtectionResourceHeaderRead(ctx, d, m)
}

func resourceProtectionResourceHeaderDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, hID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Header deleting (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	headerID, err := strconv.ParseInt(hID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	if _, err := client.Headers.Delete(ctx, resourceID, headerID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish DDoS Protection Resource Header deleting")

	return nil
}
