package lblistener

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func setAllowedCIDRs(_ context.Context, d *schema.ResourceData, listener *edgecloud.Listener) error {
	if len(listener.AllowedCIDRs) > 0 {
		allowedCIDRs := make([]string, 0, len(listener.AllowedCIDRs))
		allowedCIDRs = append(allowedCIDRs, listener.AllowedCIDRs...)
		return d.Set("allowed_cidrs", allowedCIDRs)
	}

	return nil
}

func setInsertHeaders(_ context.Context, d *schema.ResourceData, listener *edgecloud.Listener) error {
	if len(listener.InsertHeaders) > 0 {
		return d.Set("insert_headers", map[string]interface{}{
			"X-Forwarded-For":   "true",
			"X-Forwarded-Port":  "true",
			"X-Forwarded-Proto": "true",
		})
	}

	return nil
}
