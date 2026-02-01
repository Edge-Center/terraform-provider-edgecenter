package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecenteredgemon-go/channel"
)

func resourceRMONChannel() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"receiver": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Receiver type used in API path. Only 'telegram', 'slack', 'pd', 'mm', 'email' are allowed.",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The token used for the channel.",
			},
			"channel_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The channel identifier.",
			},
		},
		CreateContext: resourceChannelCreate,
		ReadContext:   resourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,
	}
}

func resourceChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Channel creating")

	cfg := m.(*Config)
	client := cfg.RmonClient

	receiver := d.Get("receiver").(string)

	req := channel.Request{
		Channel: d.Get("channel_name").(string),
		Token:   d.Get("token").(string),
	}

	resp, err := client.Channel().Create(ctx, receiver, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))

	log.Printf("[DEBUG] Finish RMON Channel creating (id=%s)\n", d.Id())
	return resourceChannelRead(ctx, d, m)
}

func resourceChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Channel reading (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}
	receiver := d.Get("receiver").(string)

	resp, err := client.Channel().Get(ctx, receiver, id)
	if err != nil {
		if isNotFoundErr(err) {
			log.Printf("[WARN] RMON Channel not found (id=%s)\n", idStr)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("receiver", receiver)

	if err := d.Set("channel_name", resp.Channel); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("token", resp.Token); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish RMON Channel reading")
	return nil
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Channel updating (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}
	receiver := d.Get("receiver").(string)

	if d.HasChange("channel_name") || d.HasChange("token") {
		req := channel.Request{
			Channel: d.Get("channel_name").(string),
			Token:   d.Get("token").(string),
		}

		if err := client.Channel().Update(ctx, receiver, id, &req); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] Finish RMON Channel updating (id=%s)\n", idStr)
	return resourceChannelRead(ctx, d, m)
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	idStr := d.Id()
	log.Printf("[DEBUG] Start RMON Channel deleting (id=%s)\n", idStr)

	cfg := m.(*Config)
	client := cfg.RmonClient

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return diag.FromErr(err)
	}
	receiver := d.Get("receiver").(string)

	if err := client.Channel().Delete(ctx, receiver, id); err != nil {
		if !isNotFoundErr(err) {
			return diag.FromErr(err)
		}
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Channel deleting")
	return nil
}
