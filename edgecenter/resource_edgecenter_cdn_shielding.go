package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercdn-go/shielding"
)

func resourceCDNShielding() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "CDN resource ID",
			},
			"shielding_pop": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "ID of the shielding pop",
			},
		},
		CreateContext: resourceCDNShieldingUpdate,
		ReadContext:   resourceCDNShieldingRead,
		UpdateContext: resourceCDNShieldingUpdate,
		DeleteContext: resourceCDNShieldingDelete,
		Description:   "Represent origin shielding",
	}
}

func resourceCDNShieldingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.Errorf("Failed reading: provided wrong resource_id: %s", d.Id())
	}

	log.Printf("[DEBUG] Start CDN Shielding reading (resource_id=%d)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	result, err := client.Shielding().Get(ctx, int64(resourceID))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("shielding_pop", result.ShieldingPop)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Finish CDN Shielding reading for (resource_id=%d)", resourceID)

	return nil
}

func resourceCDNShieldingUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get("resource_id").(int)
	log.Printf("[DEBUG] Start CDN Shielding updating for (resource_id=%d)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	var req shielding.UpdateShieldingData
	req.ShieldingPop = pointer.ToInt(d.Get("shielding_pop").(int))

	if _, err := client.Shielding().Update(ctx, int64(resourceID), &req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resourceID))
	resourceCDNShieldingRead(ctx, d, m)

	log.Printf("[DEBUG] Finish CDN Shielding updating.")

	return nil
}

func resourceCDNShieldingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get("resource_id").(int)
	log.Printf("[DEBUG] Start CDN Shielding deleting (resource_id=%d)\n", resourceID)
	config := m.(*Config)
	client := config.CDNClient

	var req shielding.UpdateShieldingData
	var intPointer *int
	req.ShieldingPop = intPointer

	if _, err := client.Shielding().Update(ctx, int64(resourceID), &req); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish CDN Origin Shielding deleting.")

	return nil
}
