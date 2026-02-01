package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecenteredgemon-go/checkgroup"
)

func resourceRMONCheckGroup() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "RMON check group name.",
			},
		},
		CreateContext: resourceCheckGroupCreate,
		ReadContext:   resourceCheckGroupRead,
		UpdateContext: resourceCheckGroupUpdate,
		DeleteContext: resourceCheckGroupDelete,
	}
}

func resourceCheckGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Check Group creating")
	config := m.(*Config)
	client := config.RmonClient

	var req checkgroup.Request
	req.Name = d.Get("name").(string)

	resp, err := client.CheckGroup().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))

	log.Printf("[DEBUG] Finish RMON Check Group creating (id=%d)\n", resp.ID)
	return resourceCheckGroupRead(ctx, d, m)
}

func resourceCheckGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Check Group reading (id=%s)\n", idStr)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.CheckGroup().Get(ctx, id)
	if err != nil {
		if isNotFoundErr(err) {
			log.Printf("[WARN] RMON Check Group not found, removing from state (id=%s)\n", idStr)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", resp.Name)

	log.Println("[DEBUG] Finish RMON Check Group reading")
	return nil
}

func resourceCheckGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Check Group updating (id=%s)\n", idStr)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		var req checkgroup.Request
		req.Name = d.Get("name").(string)

		if _, err := client.CheckGroup().Update(ctx, id, &req); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Finish RMON Check Group updating (id=%s)\n", idStr)
	return resourceCheckGroupRead(ctx, d, m)
}

func resourceCheckGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Check Group deleting (id=%s)\n", idStr)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.CheckGroup().Delete(ctx, id); err != nil {
		if !isNotFoundErr(err) {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Check Group deleting")
	return nil
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "404") ||
		strings.Contains(strings.ToLower(s), "not found")
}
