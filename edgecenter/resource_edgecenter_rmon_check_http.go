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

	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkhttp"
)

func resourceRMONCheckHTTP() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Check Http.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the Check HTTP(s).",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enabled state of the Check HTTP(s).",
			},
			"check_group": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the check group for group HTTP(s) checks.",
			},
			"place": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Port number for binding Check HTTP(s).",
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
			"url": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "URL what must be checked.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"method": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "HTTP method for HTTP(s) check.",
				ValidateFunc: validation.StringInSlice([]string{
					"get",
					"post",
					"put",
					"delete",
					"options",
					"head",
				}, false),
			},
			"ignore_ssl_error": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Ignore TLS/SSL error.",
			},
			"accepted_status_codes": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Accepted HTTP status codes (e.g. [200, 201]).",
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntBetween(100, 599),
				},
			},
			"body": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Check body answer.",
			},
			"body_req": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Send body to server. In JSON.",
				ValidateFunc: validation.StringIsJSON,
			},
			"header_req": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Send headers to server. In JSON.",
				ValidateFunc: validation.StringIsJSON,
			},
			"retries": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "Number of retries before check is marked down.",
				ValidateFunc: validation.IntAtLeast(0),
				Default:      3,
			},
			"redirects": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "Maximum number of redirects to follow. Set to 0 to disable redirects.",
				ValidateFunc: validation.IntAtLeast(0),
				Default:      3,
			},
			"runbook": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Runbook URL for alerts.",
			},
		},
		CreateContext: resourceCheckHTTPCreate,
		ReadContext:   resourceCheckHTTPRead,
		UpdateContext: resourceCheckHTTPUpdate,
		DeleteContext: resourceCheckHTTPDelete,
	}
}

func resourceCheckHTTPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start RMON Check HTTP creating")
	config := m.(*Config)
	client := config.RmonClient

	req := buildCheckHTTPRequest(d)

	resp, err := client.CheckHTTP().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", resp.ID))

	log.Printf("[DEBUG] Finish RMON Check HTTP creating (id=%d)\n", resp.ID)
	return resourceCheckHTTPRead(ctx, d, m)
}

func resourceCheckHTTPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check HTTP reading (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.CheckHTTP().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", strings.ReplaceAll(resp.Name, "'", ""))
	_ = d.Set("description", strings.ReplaceAll(resp.Description, "'", ""))
	_ = d.Set("enabled", intToBool(float64(resp.Enabled)))
	_ = d.Set("check_group", resp.CheckGroup)
	_ = d.Set("place", resp.Place)
	entities := make([]interface{}, 0, len(resp.Entities))
	if resp.Place == "all" {
		entities = []interface{}{}
	} else {
		for _, v := range resp.Entities {
			entities = append(entities, v)
		}
	}
	_ = d.Set("entities", entities)
	_ = d.Set("interval", resp.Interval)
	_ = d.Set("check_timeout", resp.CheckTimeout)
	_ = d.Set("telegram_channel_id", resp.TelegramChannelID)
	_ = d.Set("slack_channel_id", resp.SlackChannelID)
	_ = d.Set("mm_channel_id", resp.MMChannelID)
	_ = d.Set("pd_channel_id", resp.PDChannelID)
	_ = d.Set("email_channel_id", resp.EmailChannelId)
	_ = d.Set("url", resp.URL)
	_ = d.Set("method", resp.Method)
	_ = d.Set("ignore_ssl_error", resp.IgnoreSSLError)
	acceptedStatusCodes := make([]interface{}, 0, len(resp.AcceptedStatusCodes))
	for _, v := range resp.AcceptedStatusCodes {
		acceptedStatusCodes = append(acceptedStatusCodes, v)
	}
	_ = d.Set("accepted_status_codes", acceptedStatusCodes)
	_ = d.Set("body", resp.Body)
	_ = d.Set("body_req", resp.BodyReq)
	_ = d.Set("header_req", resp.HeaderReq)
	_ = d.Set("retries", resp.Retries)
	_ = d.Set("redirects", resp.Redirects)
	_ = d.Set("runbook", resp.Runbook)

	log.Println("[DEBUG] Finish RMON Check HTTP reading")
	return nil
}

func resourceCheckHTTPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check HTTP updating (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	req := buildCheckHTTPRequest(d)

	if err := client.CheckHTTP().Update(ctx, id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish RMON Check HTTP updating")
	return resourceCheckHTTPRead(ctx, d, m)
}

func resourceCheckHTTPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Id()
	log.Printf("[DEBUG] Start RMON Check HTTP deleting (id=%s)\n", resourceID)

	config := m.(*Config)
	client := config.RmonClient

	id, err := strconv.Atoi(resourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.CheckHTTP().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish RMON Check HTTP deleting")
	return nil
}

func buildCheckHTTPRequest(d *schema.ResourceData) checkhttp.Request {
	return checkhttp.Request{
		Description:         strings.ReplaceAll(d.Get("description").(string), "'", ""),
		Enabled:             boolToInt(d.Get("enabled").(bool)),
		Name:                strings.ReplaceAll(d.Get("name").(string), "'", ""),
		CheckGroup:          d.Get("check_group").(string),
		Place:               d.Get("place").(string),
		Entities:            expandIntList(d.Get("entities").([]interface{})),
		Interval:            d.Get("interval").(int),
		CheckTimeout:        d.Get("check_timeout").(int),
		TelegramChannelID:   d.Get("telegram_channel_id").(int),
		SlackChannelID:      d.Get("slack_channel_id").(int),
		MMChannelID:         d.Get("mm_channel_id").(int),
		PDChannelID:         d.Get("pd_channel_id").(int),
		EmailChannelId:      d.Get("email_channel_id").(int),
		URL:                 d.Get("url").(string),
		Method:              d.Get("method").(string),
		IgnoreSSLError:      boolToInt(d.Get("ignore_ssl_error").(bool)),
		Body:                d.Get("body").(string),
		BodyReq:             d.Get("body_req").(string),
		HeaderReq:           d.Get("header_req").(string),
		Retries:             d.Get("retries").(int),
		Redirects:           d.Get("redirects").(int),
		AcceptedStatusCodes: expandIntList(d.Get("accepted_status_codes").([]interface{})),
		Runbook:             d.Get("runbook").(string),
	}
}

func expandIntList(v []interface{}) []int {
	out := make([]int, 0, len(v))
	for _, it := range v {
		out = append(out, it.(int))
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v float64) bool {
	return int(v) == 1
}
