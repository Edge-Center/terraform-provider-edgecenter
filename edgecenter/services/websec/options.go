package websec

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/edgecentercdn-go/websec/options"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/tfutil"
)

var (
	countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

	validateTrimmed = stringValidator(
		func(value string) bool { return value == strings.TrimSpace(value) },
		"have no leading or trailing whitespace (the API trims it)",
	)
	validateLowercase = stringValidator(
		func(value string) bool { return value == strings.ToLower(value) },
		"be lowercase (the API lowercases it)",
	)
	validateCountryCode = stringValidator(
		countryCodePattern.MatchString,
		"be an ISO 3166-1 alpha-2 country code in uppercase",
	)
	validateNetwork = stringValidator(
		func(value string) bool { _, ok := normalizeNetwork(value); return ok },
		"be an IPv4 or IPv6 address or network",
	)
)

func stringValidator(valid func(string) bool, expectation string) func(interface{}, string) ([]string, []error) {
	return func(v interface{}, k string) ([]string, []error) {
		if value := v.(string); !valid(value) {
			return nil, []error{fmt.Errorf("expected %s to %s, got %q", k, expectation, value)}
		}

		return nil, nil
	}
}

func normalizeNetwork(value string) (string, bool) {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked().String(), true
	}

	if addr, err := netip.ParseAddr(value); err == nil && addr.Zone() == "" {
		return netip.PrefixFrom(addr, addr.BitLen()).String(), true
	}

	return "", false
}

func hashNetwork(v interface{}) int {
	value := v.(string)
	if network, ok := normalizeNetwork(value); ok {
		value = network
	}

	return schema.HashString(value)
}

func resourceOptionsSchema() *schema.Schema {
	return optionsSchema(
		"Protection options of the resource. The resource is created with exactly the options set here: every other option, "+
			"including the ones this provider does not manage, is sent as null, so no API defaults apply. An update sends only "+
			"the options that changed, and removing an option block deletes the option. Options this provider does not manage "+
			"are neither read nor changed after create. An empty `options {}` block is rejected.",
		resourceOptionSchemas(),
	)
}

func locationOptionsSchema() *schema.Schema {
	return optionsSchema(
		"Protection options of the location. The location is created with exactly the options set here, so no API "+
			"defaults apply. An update sends only the options that changed, and removing an option block deletes the option. "+
			"An empty `options {}` block is rejected.",
		locationOptionSchemas(),
	)
}

func optionsSchema(description string, optionSchemas map[string]*schema.Schema) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MaxItems:    1,
		Optional:    true,
		Description: description,
		Elem:        &schema.Resource{Schema: optionSchemas},
	}
}

func locationOptionSchemas() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"allowed_http_methods": listOptionSchema(
			"HTTP methods allowed in requests. The methods you can use are set per client and resource, by default "+
				"\"GET\", \"HEAD\", \"POST\", \"PUT\", \"PATCH\", \"DELETE\" and \"OPTIONS\".",
			"Allowed HTTP methods.",
			&schema.Schema{Type: schema.TypeString, ValidateFunc: validateTrimmed},
			nil,
		),
		"asn_acl": accessPolicyOptionSchema(
			"Allow or deny requests by the autonomous system number of the client.",
			"ASNs that are the exceptions from `policy_type`, from 1 to 2147483647. Private 4-byte ASNs above 2147483647 "+
				"cannot be set.",
			&schema.Schema{Type: schema.TypeInt, ValidateFunc: validation.IntBetween(1, math.MaxInt32)},
		),
		"referer_acl": accessPolicyOptionSchema(
			"Allow or deny requests by the Referer header.",
			"Domains, \"*.domain\" or \"domain.*\" patterns in lowercase that are the exceptions from `policy_type`. An "+
				"empty string stands for requests without the Referer header and needs at least one more value.",
			&schema.Schema{Type: schema.TypeString, ValidateFunc: validation.All(validateTrimmed, validateLowercase)},
		),
		"user_agent_acl": accessPolicyOptionSchema(
			"Allow or deny requests by the User-Agent header.",
			"User agents that are the exceptions from `policy_type`. A value that starts with \"~\" is a regular expression.",
			&schema.Schema{Type: schema.TypeString, ValidateFunc: validateTrimmed},
		),
		"antibot_flood_challenge": optionSchema(
			"Check that a request comes from a browser before passing it on. The check answers with HTTP 307 and sets "+
				"the antibot cookie. Disabling it on a resource also disables the antibot URL exceptions and user agent "+
				"whitelist of the resource, which this provider does not manage.",
			map[string]*schema.Schema{
				"value": {
					Type:     schema.TypeString,
					Optional: true,
					ValidateFunc: validation.StringInSlice([]string{
						"",
						string(options.AntibotChallenge),
						string(options.AntibotChallengeJS),
						string(options.AntibotChallengeJSBox),
					}, false),
					Description: "Check type: \"ANTIBOT_CHALLENGE\" checks the browser request, \"ANTIBOT_CHALLENGE_JS\" checks " +
						"it with a JavaScript file, \"ANTIBOT_CHALLENGE_JS_BOX\" needs a human interaction. Required when the " +
						"option is enabled.",
				},
				"antibot_challenge_timelimit": {
					Type:         schema.TypeInt,
					Optional:     true,
					Default:      10,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "Lifetime of the antibot cookie in minutes. Defaults to `10`.",
				},
				"antibot_challenge_get_only_on": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Apply the antibot cookie to GET requests only. Defaults to `false`.",
				},
				"antibot_challenge_utm_referrer": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Keep the UTM parameters of the original request through the challenge redirect. Defaults to `false`.",
				},
			},
		),
	}
}

func resourceOptionSchemas() map[string]*schema.Schema {
	optionSchemas := locationOptionSchemas()

	optionSchemas["blacklist"] = listOptionSchema(
		"IPv4 and IPv6 addresses and networks denied access to the resource.",
		"Addresses and networks. The API stores an address as a network, \"1.2.3.4\" as \"1.2.3.4/32\", and both forms "+
			"are the same value here.",
		&schema.Schema{Type: schema.TypeString, ValidateFunc: validateNetwork},
		hashNetwork,
	)
	optionSchemas["whitelist"] = listOptionSchema(
		"IPv4 and IPv6 addresses and networks on the whitelist of the resource.",
		"Addresses and networks. The API stores an address as a network, \"1.2.3.4\" as \"1.2.3.4/32\", and both forms "+
			"are the same value here.",
		&schema.Schema{Type: schema.TypeString, ValidateFunc: validateNetwork},
		hashNetwork,
	)
	optionSchemas["country_acl"] = accessPolicyOptionSchema(
		"Allow or deny requests by the country of the client.",
		"ISO 3166-1 alpha-2 country codes in uppercase that are the exceptions from `policy_type`.",
		&schema.Schema{Type: schema.TypeString, ValidateFunc: validateCountryCode},
	)
	optionSchemas["ip_hash"] = booleanOptionSchema("Send all requests of a client to the same source, chosen by the client IP address.")
	optionSchemas["redirect_http_to_https"] = booleanOptionSchema(
		"Redirect HTTP requests to HTTPS. The resource needs a certificate, `ssl_certificate` or one issued by " +
			"`edgecenter_websec_lecert`. A Let's Encrypt certificate is issued after the resource is created, so enable the " +
			"redirect for it in a later apply, and disable the redirect before removing the `edgecenter_websec_lecert`.",
	)
	optionSchemas["response_headers"] = optionSchema(
		"Add headers to the responses. Header names are case insensitive and must be unique. Which names and values the "+
			"API accepts is set per client.",
		map[string]*schema.Schema{
			"value": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Headers to add.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateTrimmed,
							Description:  "Header name.",
						},
						"value": {
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validateTrimmed},
							Description: "Header values.",
						},
						"always": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Add the header to every response, whatever its status code. Defaults to `false`.",
						},
					},
				},
			},
		},
	)
	optionSchemas["source_protocol_https"] = booleanOptionSchema("Connect to the sources over HTTPS.")
	optionSchemas["tls_versions"] = listOptionSchema(
		"TLS versions the resource accepts. The versions you can use are set per client, by default \"SSLv3\", \"TLSv1\", "+
			"\"TLSv1.1\", \"TLSv1.2\" and \"TLSv1.3\". Without this block the resource is created with no TLS versions "+
			"option at all, so set it explicitly.",
		"TLS versions.",
		&schema.Schema{Type: schema.TypeString, ValidateFunc: validateTrimmed},
		nil,
	)
	optionSchemas["waf"] = booleanOptionSchema("Enable the web application firewall. The WAF must be enabled on the account.")
	optionSchemas["www_redirects"] = booleanOptionSchema(
		"Redirect requests from the www subdomain to `domain`. Enabling it needs `www.<domain>` in `subdomains`. " +
			"Removing that subdomain deletes the option in the API, so remove this block in the same change.",
	)

	return optionSchemas
}

func optionSchema(description string, fields map[string]*schema.Schema) *schema.Schema {
	fields["enabled"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Enable or disable the option. Defaults to `true`.",
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		MaxItems:    1,
		Optional:    true,
		Description: description,
		Elem:        &schema.Resource{Schema: fields},
	}
}

func booleanOptionSchema(description string) *schema.Schema {
	return optionSchema(description, map[string]*schema.Schema{
		"value": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Value of the option.",
		},
	})
}

func listOptionSchema(description, valueDescription string, elem *schema.Schema, hash schema.SchemaSetFunc) *schema.Schema {
	return optionSchema(description, map[string]*schema.Schema{
		"value": {
			Type:        schema.TypeSet,
			Required:    true,
			Elem:        elem,
			Set:         hash,
			Description: valueDescription,
		},
	})
}

func accessPolicyOptionSchema(description, valuesDescription string, elem *schema.Schema) *schema.Schema {
	return optionSchema(description, map[string]*schema.Schema{
		"policy_type": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{string(options.PolicyTypeAllow), string(options.PolicyTypeDeny)}, false),
			Description:  "Default policy, \"allow\" or \"deny\". Requests that match `excepted_values` get the opposite one.",
		},
		"excepted_values": {
			Type:        schema.TypeSet,
			Required:    true,
			Elem:        elem,
			Description: valuesDescription,
		},
	})
}

func validateOptionsNotEmpty(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("options") {
		return nil
	}

	if blocks := d.Get("options").([]interface{}); len(blocks) == 1 && blocks[0] == nil {
		return errors.New("the options block is empty: set at least one option in it or remove the block")
	}

	return nil
}

func validateWWWRedirects(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("options.0.www_redirects") || !d.NewValueKnown("subdomains") || !d.NewValueKnown("domain") {
		return nil
	}

	block, ok := tfutil.GetOptByName(optionBlocks(d.Get("options")), "www_redirects")
	if !ok {
		return nil
	}

	www := "www." + d.Get("domain").(string)
	oldSubdomains, newSubdomains := d.GetChange("subdomains")
	if newSubdomains.(*schema.Set).Contains(www) {
		return nil
	}

	if block["enabled"].(bool) && block["value"].(bool) {
		return fmt.Errorf("options.0.www_redirects needs %q in subdomains", www)
	}

	if oldSubdomains.(*schema.Set).Contains(www) {
		return fmt.Errorf("removing %q from subdomains deletes options.0.www_redirects in the API: remove the block in the same change", www)
	}

	return nil
}

func validateResponseHeaderNames(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("options.0.response_headers") || !d.NewValueKnown("options.0.response_headers.0.value") {
		return nil
	}

	block, ok := tfutil.GetOptByName(optionBlocks(d.Get("options")), "response_headers")
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	for _, item := range block["value"].([]interface{}) {
		name := item.(map[string]interface{})["name"].(string)
		if name == "" {
			continue
		}

		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("options.0.response_headers has more than one %q header because names are case insensitive: put all its values into one header", name)
		}

		seen[key] = true
	}

	return nil
}

func optionBlocks(raw interface{}) map[string]interface{} {
	list := raw.([]interface{})
	if len(list) == 0 {
		return nil
	}

	blocks, _ := list[0].(map[string]interface{})

	return blocks
}

type optionsExpander struct {
	d           *schema.ResourceData
	blocks      map[string]interface{}
	onlyChanged bool
	changed     bool
}

func newOptionsExpander(d *schema.ResourceData, onlyChanged bool) *optionsExpander {
	return &optionsExpander{d: d, blocks: optionBlocks(d.Get("options")), onlyChanged: onlyChanged}
}

func expandOption[T any](e *optionsExpander, name string, expand func(map[string]interface{}) T) options.Field[T] {
	if e.onlyChanged && !e.d.HasChange("options.0."+name) {
		return options.Field[T]{}
	}

	e.changed = true

	block, ok := tfutil.GetOptByName(e.blocks, name)
	if !ok {
		return options.Null[T]()
	}

	return options.Value(expand(block))
}

func unmanagedOption[T any](e *optionsExpander) options.Field[T] {
	if e.onlyChanged {
		return options.Field[T]{}
	}

	return options.Null[T]()
}

func (e *optionsExpander) locationOptions() options.LocationOptions {
	return options.LocationOptions{
		AllowedHTTPMethods:    expandOption(e, "allowed_http_methods", expandListOption),
		ASNACL:                expandOption(e, "asn_acl", expandASNACLOption),
		RefererACL:            expandOption(e, "referer_acl", expandAccessPolicyOption),
		UserAgentACL:          expandOption(e, "user_agent_acl", expandAccessPolicyOption),
		AntibotFloodChallenge: expandOption(e, "antibot_flood_challenge", expandAntibotFloodChallengeOption),
	}
}

func expandLocationOptions(d *schema.ResourceData, onlyChanged bool) *options.LocationOptions {
	e := newOptionsExpander(d, onlyChanged)

	opts := e.locationOptions()
	if !e.changed {
		return nil
	}

	return &opts
}

func expandResourceOptions(d *schema.ResourceData, onlyChanged bool) *options.ResourceOptions {
	e := newOptionsExpander(d, onlyChanged)

	opts := options.ResourceOptions{
		LocationOptions:                         e.locationOptions(),
		UseDefaultLEChain:                       unmanagedOption[options.BooleanOption](e),
		UseRSALECert:                            unmanagedOption[options.BooleanOption](e),
		Blacklist:                               expandOption(e, "blacklist", expandListOption),
		Whitelist:                               expandOption(e, "whitelist", expandListOption),
		GeoACL:                                  unmanagedOption[options.GeoACLOption](e),
		IPHash:                                  expandOption(e, "ip_hash", expandBooleanOption),
		RedirectHTTPToHTTPS:                     expandOption(e, "redirect_http_to_https", expandBooleanOption),
		ResponseHeaders:                         expandOption(e, "response_headers", expandResponseHeadersOption),
		TLSVersions:                             expandOption(e, "tls_versions", expandListOption),
		CountryACL:                              expandOption(e, "country_acl", expandAccessPolicyOption),
		WWWRedirects:                            expandOption(e, "www_redirects", expandBooleanOption),
		SourceProtocolHTTPS:                     expandOption(e, "source_protocol_https", expandBooleanOption),
		UpstreamHealthChecker:                   unmanagedOption[options.UpstreamHealthCheckerOption](e),
		WAF:                                     expandOption(e, "waf", expandBooleanOption),
		MLAnalysis:                              unmanagedOption[options.BooleanOption](e),
		WAFMonitoringMode:                       unmanagedOption[options.BooleanOption](e),
		RedirectHTTPSToHTTP:                     unmanagedOption[options.BooleanOption](e),
		AntibotChallengeURLExceptions:           unmanagedOption[options.ListOption](e),
		AntibotFloodChallengeUserAgentWhitelist: unmanagedOption[options.ListOption](e),
	}
	if !e.changed {
		return nil
	}

	return &opts
}

func expandBooleanOption(block map[string]interface{}) options.BooleanOption {
	return options.BooleanOption{
		Enabled: block["enabled"].(bool),
		Value:   block["value"].(bool),
	}
}

func expandListOption(block map[string]interface{}) options.ListOption {
	return options.ListOption{
		Enabled: block["enabled"].(bool),
		Value:   stringsFromSet(block["value"].(*schema.Set)),
	}
}

func expandAccessPolicyOption(block map[string]interface{}) options.AccessPolicyOption {
	return options.AccessPolicyOption{
		Enabled:        block["enabled"].(bool),
		PolicyType:     options.PolicyType(block["policy_type"].(string)),
		ExceptedValues: stringsFromSet(block["excepted_values"].(*schema.Set)),
	}
}

func expandASNACLOption(block map[string]interface{}) options.ASNACLOption {
	values := block["excepted_values"].(*schema.Set).List()

	asns := make([]int64, 0, len(values))
	for _, value := range values {
		asns = append(asns, int64(value.(int)))
	}

	return options.ASNACLOption{
		Enabled:        block["enabled"].(bool),
		PolicyType:     options.PolicyType(block["policy_type"].(string)),
		ExceptedValues: asns,
	}
}

func expandResponseHeadersOption(block map[string]interface{}) options.ResponseHeadersOption {
	items := block["value"].([]interface{})

	headers := make([]options.ResponseHeader, 0, len(items))
	for _, item := range items {
		header := item.(map[string]interface{})
		headers = append(headers, options.ResponseHeader{
			Name:   header["name"].(string),
			Value:  stringsFromSet(header["value"].(*schema.Set)),
			Always: header["always"].(bool),
		})
	}

	return options.ResponseHeadersOption{
		Enabled: block["enabled"].(bool),
		Value:   headers,
	}
}

func expandAntibotFloodChallengeOption(block map[string]interface{}) options.AntibotFloodChallengeOption {
	opt := options.AntibotFloodChallengeOption{
		Enabled:     block["enabled"].(bool),
		TimeLimit:   block["antibot_challenge_timelimit"].(int),
		GetOnlyOn:   block["antibot_challenge_get_only_on"].(bool),
		UTMReferrer: block["antibot_challenge_utm_referrer"].(bool),
	}

	if value := block["value"].(string); value != "" {
		challenge := options.AntibotChallengeType(value)
		opt.Value = &challenge
	}

	return opt
}

func stringsFromSet(set *schema.Set) []string {
	values := make([]string, 0, set.Len())
	for _, value := range set.List() {
		values = append(values, value.(string))
	}

	return values
}

func flattenLocationOptions(opts *options.LocationOptions) ([]interface{}, error) {
	blocks := make(map[string]interface{})
	if opts != nil {
		if err := flattenLocationOptionBlocks(blocks, opts); err != nil {
			return nil, err
		}
	}

	return optionsList(blocks), nil
}

func flattenResourceOptions(opts *options.ResourceOptions) ([]interface{}, error) {
	blocks := make(map[string]interface{})
	if opts != nil {
		if err := flattenLocationOptionBlocks(blocks, &opts.LocationOptions); err != nil {
			return nil, err
		}

		flattenOption(blocks, "blacklist", opts.Blacklist, flattenListOption)
		flattenOption(blocks, "whitelist", opts.Whitelist, flattenListOption)
		flattenOption(blocks, "ip_hash", opts.IPHash, flattenBooleanOption)
		flattenOption(blocks, "redirect_http_to_https", opts.RedirectHTTPToHTTPS, flattenBooleanOption)
		flattenOption(blocks, "response_headers", opts.ResponseHeaders, flattenResponseHeadersOption)
		flattenOption(blocks, "tls_versions", opts.TLSVersions, flattenListOption)
		flattenOption(blocks, "country_acl", opts.CountryACL, flattenAccessPolicyOption)
		flattenOption(blocks, "www_redirects", opts.WWWRedirects, flattenBooleanOption)
		flattenOption(blocks, "source_protocol_https", opts.SourceProtocolHTTPS, flattenBooleanOption)
		flattenOption(blocks, "waf", opts.WAF, flattenBooleanOption)
	}

	return optionsList(blocks), nil
}

func flattenLocationOptionBlocks(blocks map[string]interface{}, opts *options.LocationOptions) error {
	flattenOption(blocks, "allowed_http_methods", opts.AllowedHTTPMethods, flattenListOption)
	flattenOption(blocks, "referer_acl", opts.RefererACL, flattenAccessPolicyOption)
	flattenOption(blocks, "user_agent_acl", opts.UserAgentACL, flattenAccessPolicyOption)
	flattenOption(blocks, "antibot_flood_challenge", opts.AntibotFloodChallenge, flattenAntibotFloodChallengeOption)

	if asnACL, ok := opts.ASNACL.Get(); ok {
		block, err := flattenASNACLOption(asnACL)
		if err != nil {
			return err
		}

		blocks["asn_acl"] = []interface{}{block}
	}

	return nil
}

func optionsList(blocks map[string]interface{}) []interface{} {
	if len(blocks) == 0 {
		return []interface{}{}
	}

	return []interface{}{blocks}
}

func flattenOption[T any](blocks map[string]interface{}, name string, field options.Field[T], flatten func(T) map[string]interface{}) {
	if value, ok := field.Get(); ok {
		blocks[name] = []interface{}{flatten(value)}
	}
}

func flattenBooleanOption(opt options.BooleanOption) map[string]interface{} {
	return map[string]interface{}{
		"enabled": opt.Enabled,
		"value":   opt.Value,
	}
}

func flattenListOption(opt options.ListOption) map[string]interface{} {
	return map[string]interface{}{
		"enabled": opt.Enabled,
		"value":   opt.Value,
	}
}

func flattenAccessPolicyOption(opt options.AccessPolicyOption) map[string]interface{} {
	return map[string]interface{}{
		"enabled":         opt.Enabled,
		"policy_type":     string(opt.PolicyType),
		"excepted_values": opt.ExceptedValues,
	}
}

func flattenASNACLOption(opt options.ASNACLOption) (map[string]interface{}, error) {
	asns := make([]int, 0, len(opt.ExceptedValues))
	for _, asn := range opt.ExceptedValues {
		if asn > math.MaxInt32 {
			return nil, fmt.Errorf("asn_acl holds ASN %d, but Terraform accepts ASNs up to %d only", asn, math.MaxInt32)
		}

		asns = append(asns, int(asn))
	}

	return map[string]interface{}{
		"enabled":         opt.Enabled,
		"policy_type":     string(opt.PolicyType),
		"excepted_values": asns,
	}, nil
}

func flattenResponseHeadersOption(opt options.ResponseHeadersOption) map[string]interface{} {
	headers := make([]interface{}, 0, len(opt.Value))
	for _, header := range opt.Value {
		headers = append(headers, map[string]interface{}{
			"name":   header.Name,
			"value":  header.Value,
			"always": header.Always,
		})
	}

	return map[string]interface{}{
		"enabled": opt.Enabled,
		"value":   headers,
	}
}

func flattenAntibotFloodChallengeOption(opt options.AntibotFloodChallengeOption) map[string]interface{} {
	value := ""
	if opt.Value != nil {
		value = string(*opt.Value)
	}

	return map[string]interface{}{
		"enabled":                        opt.Enabled,
		"value":                          value,
		"antibot_challenge_timelimit":    opt.TimeLimit,
		"antibot_challenge_get_only_on":  opt.GetOnlyOn,
		"antibot_challenge_utm_referrer": opt.UTMReferrer,
	}
}
