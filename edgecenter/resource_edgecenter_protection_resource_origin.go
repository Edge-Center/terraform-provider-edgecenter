package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
)

const (
	modePrimary = "primary"
	modeBackup  = "backup"
	modeDown    = "down"
)

func resourceProtectionResourceOrigin() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceOriginCreate,
		ReadContext:   resourceProtectionResourceOriginRead,
		UpdateContext: resourceProtectionResourceOriginUpdate,
		DeleteContext: resourceProtectionResourceOriginDelete,
		Description:   "Represent IP address behind DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Custom comment about the origin.",
			},
			"fail_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Time (in seconds) after which the server is considered unreachable.",
			},
			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Origin IP for the website behind DDoS protection.",
			},
			"max_fails": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Max number of failed connection attempts.",
			},
			"mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: fmt.Sprintf("Operation mode for this origin. Available values are `%s`, `%s`, `%s`.", modePrimary, modeBackup, modeDown),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch v {
					case modePrimary, modeBackup, modeDown:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is `%s`, `%s`, `%s`.", v, modePrimary, modeBackup, modeDown)
				},
			},
			"resource": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the DDoS protection resource using this origin.",
			},
			"weight": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Origin weight for request balancing.",
			},
		},
	}
}

func resourceProtectionResourceOriginCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Protection Resource Origin creating")
	config := m.(*Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(d.Get("resource").(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.OriginCreateRequest

	req.IP = d.Get("ip").(string)

	if mode, ok := d.GetOk("mode"); ok {
		req.Mode = mode.(string)
	}

	if weight, ok := d.GetOk("weight"); ok {
		req.Weight = weight.(int)
	}

	if max_fails, ok := d.GetOk("max_fails"); ok {
		req.MaxFails = max_fails.(int)
	}

	if fail_timeout, ok := d.GetOk("fail_timeout"); ok {
		req.FailTimeout = fail_timeout.(int)
	}

	if comment, ok := d.GetOk("comment"); ok {
		req.Comment = comment.(string)
	}

	result, _, err := client.Origins.Create(ctx, resourceID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceOriginRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Protection Resource Origin creating (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceOriginRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Origin reading (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	originID, err := strconv.ParseInt(oID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	result, _, err := client.Origins.Get(ctx, resourceID, originID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("resource", fmt.Sprintf("%d", resourceID))
	d.Set("ip", result.IP)
	d.Set("mode", result.Mode)
	d.Set("weight", result.Weight)
	d.Set("max_fails", result.MaxFails)
	d.Set("fail_timeout", result.FailTimeout)
	d.Set("comment", result.Comment)

	log.Println("[DEBUG] Finish Protection Resource Origin reading")

	return nil
}

func resourceProtectionResourceOriginUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Origin updating (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	originID, err := strconv.ParseInt(oID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	var req protectionSDK.OriginCreateRequest

	req.IP = d.Get("ip").(string)

	if mode, ok := d.GetOk("mode"); ok {
		req.Mode = mode.(string)
	}

	if weight, ok := d.GetOk("weight"); ok {
		req.Weight = weight.(int)
	}

	if max_fails, ok := d.GetOk("max_fails"); ok {
		req.MaxFails = max_fails.(int)
	}

	if fail_timeout, ok := d.GetOk("fail_timeout"); ok {
		req.FailTimeout = fail_timeout.(int)
	}

	if comment, ok := d.GetOk("comment"); ok {
		req.Comment = comment.(string)
	}

	if _, _, err := client.Origins.Update(ctx, resourceID, originID, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Protection Resource Origin updating")

	return resourceProtectionResourceOriginRead(ctx, d, m)
}

func resourceProtectionResourceOriginDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start Protection Resource Origin deleting (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	originID, err := strconv.ParseInt(oID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	if _, err := client.Origins.Delete(ctx, resourceID, originID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish Protection Resource Origin deleting")

	return nil
}
