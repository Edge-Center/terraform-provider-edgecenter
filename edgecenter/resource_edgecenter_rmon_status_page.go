package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecenteredgemon-go/statuspage"
)

func resourceRMONStatusPage() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the status page.",
			},
			"slug": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The slug of the status page.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the status page.",
			},
			"custom_style": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom CSS style for the status page.",
			},
			"checks": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of checks associated with the status page.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"check_id": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Check ID.",
						},
					},
				},
			},
		},
		CreateContext: resourceStatusPageCreate,
		ReadContext:   resourceStatusPageRead,
		UpdateContext: resourceStatusPageUpdate,
		DeleteContext: resourceStatusPageDelete,
	}
}

func resourceStatusPageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Status Page creating")

	cfg := m.(*Config)
	client := cfg.RmonClient

	req := expandStatusPageRequest(d)

	resp, err := client.StatusPage().Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))

	log.Print("[DEBUG] Finish RMON Status Page creating")

	return resourceStatusPageRead(ctx, d, m)
}

func resourceStatusPageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Status Page reading (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.StatusPage().Get(ctx, id)
	if err != nil {
		if isNotFoundErr(err) {
			log.Printf("[WARN] RMON Status Page not found, removing from state (id=%s)\n", idStr)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if err := d.Set("name", resp.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("slug", resp.Slug); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", resp.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("custom_style", resp.CustomStyle); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("checks", flattenStatusPageChecks(resp.Checks)); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish RMON Status Page reading")

	return nil
}

func resourceStatusPageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Status Page updating (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "slug", "description", "custom_style", "checks") {
		var req statuspage.Request
		req.Name = d.Get("name").(string)
		req.Slug = d.Get("slug").(string)
		req.Description = d.Get("description").(string)
		req.CustomStyle = d.Get("custom_style").(string)

		req.Checks = expandStatusPageChecks(d.Get("checks").([]interface{}))

		if err := client.StatusPage().Update(ctx, id, &req); err != nil {
			return diag.FromErr(err)
		}
	}
	log.Printf("[DEBUG] Finish RMON Status Page updating (id=%s)\n", idStr)

	return resourceStatusPageRead(ctx, d, m)
}

func resourceStatusPageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Status Page deleting (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.StatusPage().Delete(ctx, id); err != nil {
		if !isNotFoundErr(err) {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Status Page deleting")

	return nil
}

func expandStatusPageRequest(d *schema.ResourceData) *statuspage.Request {
	req := &statuspage.Request{
		Base: statuspage.Base{
			Name:        d.Get("name").(string),
			Slug:        d.Get("slug").(string),
			Description: d.Get("description").(string),
			CustomStyle: d.Get("custom_style").(string),
		},
		Checks: expandStatusPageChecks(d.Get("checks").([]interface{})),
	}

	return req
}

func expandStatusPageChecks(raw []interface{}) []int {
	out := make([]int, 0, len(raw))
	for _, v := range raw {
		m := v.(map[string]interface{})
		out = append(out, m["check_id"].(int))
	}

	return out
}

func flattenStatusPageChecks(in []statuspage.Checks) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(in))
	for _, c := range in {
		out = append(out, map[string]interface{}{
			"check_id": c.CheckID,
		})
	}

	return out
}
