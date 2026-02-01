package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkdns"
)

func resourceRMONCheckDNS() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Check DNS.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the Check DNS.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enabled state of the Check DNS.",
			},
			"check_group": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the check group for group DNS checks.",
			},
			"place": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Place scope for Check DNS.",
				ValidateFunc: validation.StringInSlice([]string{
					"all",
					"country",
					"region",
					"agent",
				}, false),
			},
			"entities": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of entities where check must be created.",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"interval": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Interval in seconds between checks.",
				Default:     120,
			},
			"check_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Answer timeout in seconds.",
				Default:     2,
			},
			"telegram_channel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Telegram channel ID for alerts.",
			},
			"slack_channel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Slack channel ID for alerts.",
			},
			"mm_channel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Mattermost channel ID for alerts.",
			},
			"pd_channel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "PagerDuty channel ID for alerts.",
			},
			"email_channel_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Email channel ID (optional)",
			},
			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "IP address or domain name for check.",
			},
			"port": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "DNS server port.",
				ValidateFunc: validation.IsPortNumber,
				Default:      53,
			},
			"resolver": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DNS server where resolve DNS query.",
			},
			"record_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DNS record type.",
				ValidateFunc: validation.StringInSlice([]string{
					"a",
					"aaa",
					"caa",
					"cname",
					"mx",
					"ns",
					"ptr",
					"sao",
					"src",
					"txt",
				}, false),
			},
			"retries": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "Number of retries before check is marked down.",
				ValidateFunc: validation.IntAtLeast(0),
				Default:      3,
			},
			"runbook": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Runbook URL for alerts.",
			},
		},
		CreateContext: resourceCheckDNSCreate,
		ReadContext:   resourceCheckDNSRead,
		UpdateContext: resourceCheckDNSUpdate,
		DeleteContext: resourceCheckDNSDelete,
	}
}

func resourceCheckDNSCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Check DNS creating")
	config := m.(*Config)
	client := config.RmonClient

	req := buildCheckDNSRequest(d)

	resp, err := client.CheckDNS().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))

	log.Printf("[DEBUG] Finish RMON Check DNS creating (id=%d)\n", resp.ID)
	return resourceCheckDNSRead(ctx, d, m)
}

func resourceCheckDNSRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check DNS reading (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.CheckDNS().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", strings.ReplaceAll(resp.Name, "'", ""))
	_ = d.Set("description", strings.ReplaceAll(resp.Description, "'", ""))
	_ = d.Set("enabled", intToBool(float64(resp.Enabled)))
	_ = d.Set("check_group", resp.CheckGroup)
	_ = d.Set("place", resp.Place)
	entities := make([]interface{}, 0, len(resp.Entities))
	for _, v := range resp.Entities {
		entities = append(entities, v)
	}
	if resp.Place == "all" {
		entities = []interface{}{}
	}
	_ = d.Set("entities", entities)
	_ = d.Set("interval", resp.Interval)
	_ = d.Set("check_timeout", resp.CheckTimeout)
	_ = d.Set("telegram_channel_id", resp.TelegramChannelID)
	_ = d.Set("slack_channel_id", resp.SlackChannelID)
	_ = d.Set("mm_channel_id", resp.MMChannelID)
	_ = d.Set("pd_channel_id", resp.PDChannelID)
	_ = d.Set("email_channel_id", resp.EmailChannelId)
	_ = d.Set("ip", resp.IP)
	_ = d.Set("port", resp.Port)
	_ = d.Set("resolver", resp.Resolver)
	_ = d.Set("record_type", resp.RecordType)
	_ = d.Set("retries", resp.Retries)
	_ = d.Set("runbook", resp.Runbook)

	log.Println("[DEBUG] Finish RMON Check DNS reading")
	return nil
}

func resourceCheckDNSUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check DNS updating (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildCheckDNSRequest(d)

	if err := client.CheckDNS().Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish RMON Check DNS updating")
	return resourceCheckDNSRead(ctx, d, m)
}

func resourceCheckDNSDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check DNS deleting (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.CheckDNS().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Check DNS deleting")
	return nil
}

func buildCheckDNSRequest(d *schema.ResourceData) checkdns.Request {
	return checkdns.Request{
		Description:       strings.ReplaceAll(d.Get("description").(string), "'", ""),
		Enabled:           boolToInt(d.Get("enabled").(bool)),
		Name:              strings.ReplaceAll(d.Get("name").(string), "'", ""),
		CheckGroup:        d.Get("check_group").(string),
		Place:             d.Get("place").(string),
		Entities:          expandIntList(d.Get("entities").([]interface{})),
		Interval:          d.Get("interval").(int),
		CheckTimeout:      d.Get("check_timeout").(int),
		TelegramChannelID: d.Get("telegram_channel_id").(int),
		SlackChannelID:    d.Get("slack_channel_id").(int),
		MMChannelID:       d.Get("mm_channel_id").(int),
		PDChannelID:       d.Get("pd_channel_id").(int),
		EmailChannelId:    d.Get("email_channel_id").(int),
		IP:                d.Get("ip").(string),
		Port:              d.Get("port").(int),
		Resolver:          d.Get("resolver").(string),
		RecordType:        d.Get("record_type").(string),
		Retries:           d.Get("retries").(int),
		Runbook:           d.Get("runbook").(string),
	}
}
