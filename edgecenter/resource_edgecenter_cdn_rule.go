package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

    cdn "github.com/Edge-Center/edgecentercdn-go/edgecenter"
	"github.com/Edge-Center/edgecentercdn-go/rules"
)

func listToLocationOptions(l []interface{}) *cdn.LocationOptions {
	if len(l) == 0 {
		return nil
	}

	var opts cdn.LocationOptions
	fields := l[0].(map[string]interface{})
	if opt, ok := getOptByName(fields, "edge_cache_settings"); ok {
		rawCustomVals := opt["custom_values"].(map[string]interface{})
		customVals := make(map[string]string, len(rawCustomVals))
		for key, value := range rawCustomVals {
			customVals[key] = value.(string)
		}

		opts.EdgeCacheSettings = &cdn.EdgeCacheSettings{
			Enabled:      opt["enabled"].(bool),
			Value:        opt["value"].(string),
			CustomValues: customVals,
			Default:      opt["default"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "browser_cache_settings"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.BrowserCacheSettings = &cdn.BrowserCacheSettings{
			Enabled: enabled,
			Value:   opt["value"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "host_header"); ok {
		opts.HostHeader = &cdn.HostHeader{
			Enabled: opt["enabled"].(bool),
			Value:   opt["value"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "redirect_http_to_https"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.RedirectHttpToHttps = &cdn.RedirectHttpToHttps{
			Enabled: enabled,
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "gzip_on"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.GzipOn = &cdn.GzipOn{
			Enabled: enabled,
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "cors"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.Cors = &cdn.Cors{
			Enabled: enabled,
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.Cors.Value = append(opts.Cors.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "rewrite"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.Rewrite = &cdn.Rewrite{
			Enabled: enabled,
			Body:    opt["body"].(string),
			Flag:    opt["flag"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "sni"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.SNI = &cdn.SNIOption{
			Enabled:        enabled,
			SNIType:        opt["sni_type"].(string),
			CustomHostname: opt["custom_hostname"].(string),
		}
	}
	if opt, ok := getOptByName(fields, "ignore_query_string"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.IgnoreQueryString = &cdn.IgnoreQueryString{
			Enabled: enabled,
			Value:   opt["value"].(bool),
		}
	}
	if opt, ok := getOptByName(fields, "query_params_whitelist"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.QueryParamsWhitelist = &cdn.QueryParamsWhitelist{
			Enabled: enabled,
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.QueryParamsWhitelist.Value = append(opts.QueryParamsWhitelist.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "query_params_blacklist"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.QueryParamsBlacklist = &cdn.QueryParamsBlacklist{
			Enabled: enabled,
		}
		for _, v := range opt["value"].(*schema.Set).List() {
			opts.QueryParamsBlacklist.Value = append(opts.QueryParamsBlacklist.Value, v.(string))
		}
	}
	if opt, ok := getOptByName(fields, "static_request_headers"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.StaticRequestHeaders = &cdn.StaticRequestHeaders{
			Enabled: enabled,
			Value:   map[string]string{},
		}
		for k, v := range opt["value"].(map[string]interface{}) {
			opts.StaticRequestHeaders.Value[k] = v.(string)
		}
	}
	if opt, ok := getOptByName(fields, "static_headers"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.StaticHeaders = &cdn.StaticHeaders{
			Enabled: enabled,
			Value:   map[string]string{},
		}
		for k, v := range opt["value"].(map[string]interface{}) {
			opts.StaticHeaders.Value[k] = v.(string)
		}
	}
	if opt, ok := getOptByName(fields, "websockets"); ok {
		enabled := true
		if _, ok := opt["enabled"]; ok {
			enabled = opt["enabled"].(bool)
		}
		opts.WebSockets = &cdn.WebSockets{
			Enabled: enabled,
			Value:   opt["value"].(bool),
		}
	}
	return &opts
}

func resourceCDNRule() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"resource_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Rule name",
			},
			"rule": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A pattern that defines when the rule is triggered. By default, we add a leading forward slash to any rule pattern. Specify a pattern without a forward slash.",
			},
			"origin_group": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "ID of the Origins Group. Use one of your Origins Group or create a new one. You can use either 'origin' parameter or 'originGroup' in the resource definition.",
			},
			"origin_protocol": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "This option defines the protocol that will be used by CDN servers to request content from an origin source. If not specified, it will be inherit from resource. Possible values are: HTTPS, HTTP, MATCH.",
			},
			"options": optionsSchema,
		},
		CreateContext: resourceCDNRuleCreate,
		ReadContext:   resourceCDNRuleRead,
		UpdateContext: resourceCDNRuleUpdate,
		DeleteContext: resourceCDNRuleDelete,
		Description:   "Represent cdn resource rule",
	}
}

func resourceCDNRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start CDN Rule creating")
	config := m.(*Config)
	client := config.CDNClient

	var req rules.CreateRequest
	req.Name = d.Get("name").(string)
	req.Rule = d.Get("rule").(string)

	if d.Get("origin_group") != nil && d.Get("origin_group").(int) > 0 {
		req.OriginGroup = pointer.ToInt(d.Get("origin_group").(int))
	}

	if d.Get("origin_protocol") != nil && d.Get("origin_protocol") != "" {
		req.OverrideOriginProtocol = pointer.ToString(d.Get("origin_protocol").(string))
	}

	resourceID := d.Get("resource_id").(int)

	req.LocationOptions = listToLocationOptions(d.Get("options").([]interface{}))

	result, err := client.Rules().Create(ctx, int64(resourceID), &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	resourceCDNRuleRead(ctx, d, m)

	log.Printf("[DEBUG] Finish CDN Rule creating (id=%d)\n", result.ID)

	return nil
}

func resourceCDNRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ruleID := d.Id()
	log.Printf("[DEBUG] Start CDN Rule reading (id=%s)\n", ruleID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(ruleID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID := d.Get("resource_id").(int)

	result, err := client.Rules().Get(ctx, int64(resourceID), id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", result.Name)
	d.Set("rule", result.Pattern)
	d.Set("origin_group", result.OriginGroup)
	d.Set("origin_protocol", result.OriginProtocol)
	if err := d.Set("options", optionsToList(result.Options)); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish CDN Rule reading")

	return nil
}

func resourceCDNRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ruleID := d.Id()
	log.Printf("[DEBUG] Start CDN Rule updating (id=%s)\n", ruleID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(ruleID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req rules.UpdateRequest
	req.Name = d.Get("name").(string)
	req.Rule = d.Get("rule").(string)

	if d.Get("origin_group") != nil && d.Get("origin_group").(int) > 0 {
		req.OriginGroup = pointer.ToInt(d.Get("origin_group").(int))
	}

	if d.Get("origin_protocol") != nil && d.Get("origin_protocol") != "" {
		req.OverrideOriginProtocol = pointer.ToString(d.Get("origin_protocol").(string))
	}

	req.Options = listToLocationOptions(d.Get("options").([]interface{}))

	resourceID := d.Get("resource_id").(int)

	if _, err := client.Rules().Update(ctx, int64(resourceID), id, &req); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish CDN Rule updating")

	return resourceCDNRuleRead(ctx, d, m)
}

func resourceCDNRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ruleID := d.Id()
	log.Printf("[DEBUG] Start CDN Rule deleting (id=%s)\n", ruleID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(ruleID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID := d.Get("resource_id").(int)

	if err := client.Rules().Delete(ctx, int64(resourceID), id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish CDN Rule deleting")

	return nil
}
