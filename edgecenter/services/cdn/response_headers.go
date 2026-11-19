package cdn

import (
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var alwaysShownResponseHeaders = []string{"connection", "content-length", "content-type", "date", "server"}

func hashHeaderName(v interface{}) int {
	return schema.HashString(strings.ToLower(v.(string)))
}

func dropUnlistedAlwaysShownHeaders(options []interface{}, d *schema.ResourceData) {
	if len(options) == 0 {
		return
	}

	opts, _ := options[0].(map[string][]interface{})
	policies := opts["response_headers_hiding_policy"]
	if len(policies) == 0 {
		return
	}

	policy, _ := policies[0].(map[string]interface{})
	headers, _ := policy["excepted"].([]string)
	listed, _ := d.Get("options.0.response_headers_hiding_policy.0.excepted").(*schema.Set)

	excepted := make([]string, 0, len(headers))
	for _, header := range headers {
		if slices.Contains(alwaysShownResponseHeaders, strings.ToLower(header)) && (listed == nil || !listed.Contains(header)) {
			continue
		}
		excepted = append(excepted, header)
	}
	policy["excepted"] = excepted
}
