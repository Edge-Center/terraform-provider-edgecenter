package protection

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	modePrimary = "primary"
	modeBackup  = "backup"
	modeDown    = "down"
)

const (
	ProtectionOriginResource = "edgecenter_protection_resource_origin"

	ProtectionOriginSchemaComment     = "comment"
	ProtectionOriginSchemaFailTimeout = "fail_timeout"
	ProtectionOriginSchemaIP          = "ip"
	ProtectionOriginSchemaMaxFails    = "max_fails"
	ProtectionOriginSchemaMode        = "mode"
	ProtectionOriginSchemaResource    = "resource"
	ProtectionOriginSchemaWeight      = "weight"
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
			ProtectionOriginSchemaComment: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Custom comment about the origin.",
			},
			ProtectionOriginSchemaFailTimeout: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Time (in seconds) after which the server is considered unreachable.",
			},
			ProtectionOriginSchemaIP: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Origin IP for the website behind DDoS protection.",
			},
			ProtectionOriginSchemaMaxFails: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Max number of failed connection attempts.",
			},
			ProtectionOriginSchemaMode: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  fmt.Sprintf("Operation mode for this origin. Available values are `%s`, `%s`, `%s`.", modePrimary, modeBackup, modeDown),
				ValidateFunc: validation.StringInSlice([]string{modePrimary, modeBackup, modeDown}, false),
			},
			ProtectionOriginSchemaResource: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the DDoS protection resource using this origin.",
			},
			ProtectionOriginSchemaWeight: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Origin weight for request balancing.",
			},
		},
	}
}

func resourceProtectionResourceOriginCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start DDoS Protection Resource Origin creating")
	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	resourceID, err := strconv.ParseInt(d.Get(ProtectionOriginSchemaResource).(string), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.OriginCreateRequest

	req.IP = d.Get(ProtectionOriginSchemaIP).(string)

	if mode, ok := d.GetOk(ProtectionOriginSchemaMode); ok {
		req.Mode = mode.(string)
	}

	if weight, ok := d.GetOk(ProtectionOriginSchemaWeight); ok {
		req.Weight = weight.(int)
	}

	if max_fails, ok := d.GetOk(ProtectionOriginSchemaMaxFails); ok {
		req.MaxFails = max_fails.(int)
	}

	if fail_timeout, ok := d.GetOk(ProtectionOriginSchemaFailTimeout); ok {
		req.FailTimeout = fail_timeout.(int)
	}

	if comment, ok := d.GetOk(ProtectionOriginSchemaComment); ok {
		req.Comment = comment.(string)
	}

	result, _, err := client.Origins.Create(ctx, resourceID, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d:%d", resourceID, result.ID))
	resourceProtectionResourceOriginRead(ctx, d, m)

	log.Printf("[DEBUG] Finish DDoS Protection Resource Origin creating (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceOriginRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Origin reading (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	result, _, err := client.Origins.Get(ctx, resourceID, originID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(ProtectionOriginSchemaResource, fmt.Sprintf("%d", resourceID))
	d.Set(ProtectionOriginSchemaIP, result.IP)
	d.Set(ProtectionOriginSchemaMode, result.Mode)
	d.Set(ProtectionOriginSchemaWeight, result.Weight)
	d.Set(ProtectionOriginSchemaMaxFails, result.MaxFails)
	d.Set(ProtectionOriginSchemaFailTimeout, result.FailTimeout)
	d.Set(ProtectionOriginSchemaComment, result.Comment)

	log.Println("[DEBUG] Finish DDoS Protection Resource Origin reading")

	return nil
}

func resourceProtectionResourceOriginUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Origin updating (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	var req protectionSDK.OriginCreateRequest

	req.IP = d.Get(ProtectionOriginSchemaIP).(string)

	if mode, ok := d.GetOk(ProtectionOriginSchemaMode); ok {
		req.Mode = mode.(string)
	}

	if weight, ok := d.GetOk(ProtectionOriginSchemaWeight); ok {
		req.Weight = weight.(int)
	}

	if max_fails, ok := d.GetOk(ProtectionOriginSchemaMaxFails); ok {
		req.MaxFails = max_fails.(int)
	}

	if fail_timeout, ok := d.GetOk(ProtectionOriginSchemaFailTimeout); ok {
		req.FailTimeout = fail_timeout.(int)
	}

	if comment, ok := d.GetOk(ProtectionOriginSchemaComment); ok {
		req.Comment = comment.(string)
	}

	if _, _, err := client.Origins.Update(ctx, resourceID, originID, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish DDoS Protection Resource Origin updating")

	return resourceProtectionResourceOriginRead(ctx, d, m)
}

func resourceProtectionResourceOriginDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, oID, err := edgecenter.ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start DDoS Protection Resource Origin deleting (id=%s)\n", d.Id())
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

	config := m.(*edgecenter.Config)
	client := config.ProtectionClient

	if _, err := client.Origins.Delete(ctx, resourceID, originID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish DDoS Protection Resource Origin deleting")

	return nil
}
