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

	"github.com/Edge-Center/edgecenteredgemon-go/checks/checktcp"
)

func resourceRMONCheckTCP() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Check TCP.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the Check TCP.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enabled state of the Check TCP.",
			},
			"check_group": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the check group for group TCP checks.",
			},
			"place": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Place scope for Check TCP.",
				ValidateFunc: validation.StringInSlice([]string{
					"all",
					"country",
					"region",
					"agent",
				}, false),
			},
			"priority": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Where checks must be deployed.",
				ValidateFunc: validation.StringInSlice([]string{
					"info",
					"warning",
					"error",
					"critical",
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
				Description: "IP address or domain name for TCP check.",
			},
			"port": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "Port for TCP check.",
				ValidateFunc: validation.IsPortNumber,
			},
			"retries": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "Number of retries before check is marked down.",
				Default:      3,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"runbook": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Runbook URL for alerts.",
			},
		},

		CreateContext: resourceCheckTCPCreate,
		ReadContext:   resourceCheckTCPRead,
		UpdateContext: resourceCheckTCPUpdate,
		DeleteContext: resourceCheckTCPDelete,
	}
}

func resourceCheckTCPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Check TCP creating")
	config := m.(*Config)
	client := config.RmonClient

	req := buildCheckTCPRequest(d)

	resp, err := client.CheckTCP().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))
	log.Printf("[DEBUG] Finish RMON Check TCP creating (id=%d)\n", resp.ID)
	return resourceCheckTCPRead(ctx, d, m)
}

func resourceCheckTCPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check TCP reading (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.CheckTCP().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", strings.ReplaceAll(resp.Name, "'", ""))
	_ = d.Set("description", strings.ReplaceAll(resp.Description, "'", ""))
	_ = d.Set("check_group", resp.CheckGroup)
	_ = d.Set("enabled", intToBool(float64(resp.Enabled)))
	_ = d.Set("place", resp.Place)
	entities := make([]interface{}, 0, len(resp.Entities))
	for _, v := range resp.Entities {
		entities = append(entities, v)
	}
	if resp.Place == "all" {
		entities = []interface{}{}
	}
	_ = d.Set("entities", entities)
	_ = d.Set("priority", resp.Priority)
	_ = d.Set("interval", resp.Interval)
	_ = d.Set("check_timeout", resp.CheckTimeout)
	_ = d.Set("telegram_channel_id", resp.TelegramChannelID)
	_ = d.Set("slack_channel_id", resp.SlackChannelID)
	_ = d.Set("mm_channel_id", resp.MMChannelID)
	_ = d.Set("pd_channel_id", resp.PDChannelID)
	_ = d.Set("email_channel_id", resp.EmailChannelId)
	_ = d.Set("ip", resp.IP)
	_ = d.Set("port", resp.Port)
	_ = d.Set("retries", resp.Retries)
	_ = d.Set("runbook", resp.Runbook)

	log.Println("[DEBUG] Finish RMON Check TCP reading")
	return nil
}

func resourceCheckTCPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check TCP updating (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildCheckTCPRequest(d)

	if err := client.CheckTCP().Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish RMON Check TCP updating")
	return resourceCheckTCPRead(ctx, d, m)
}

func resourceCheckTCPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check TCP deleting (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.CheckTCP().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Check TCP deleting")
	return nil
}

func buildCheckTCPRequest(d *schema.ResourceData) checktcp.Request {
	return checktcp.Request{
		Description:       strings.ReplaceAll(d.Get("description").(string), "'", ""),
		Enabled:           boolToInt(d.Get("enabled").(bool)),
		Name:              strings.ReplaceAll(d.Get("name").(string), "'", ""),
		CheckGroup:        d.Get("check_group").(string),
		Place:             d.Get("place").(string),
		Priority:          d.Get("priority").(string),
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
		Retries:           d.Get("retries").(int),
		Runbook:           d.Get("runbook").(string),
	}
}
