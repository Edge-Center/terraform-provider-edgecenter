package edgecenter

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func GetLBL7Policy(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.L7Policy, error) {
	l7PolicyID, idOk := d.GetOk("id")

	var l7Policy *edgecloudV2.L7Policy
	var err error
	switch idOk {
	case true:
		l7Policy, _, err = client.L7Policies.Get(ctx, l7PolicyID.(string))
		if err != nil {
			return nil, err
		}
	default:
		lPolicyName, nameOk := d.GetOk("name")
		if nameOk {
			l7Policy, err = utilV2.GetLbL7PolicyFromName(ctx, client, lPolicyName.(string))
			if err != nil {
				switch {
				case errors.Is(err, edgecloudV2.ErrMultipleResourcesWithTheSameName):
					return nil, fmt.Errorf("%w. Use \"id\" attribute instead of \"name\"", err)
				case errors.Is(err, edgecloudV2.ErrResourceDoesntExist):
					return nil, fmt.Errorf("%w. Check if the name is correct, or try to use \"id\" attribute", err)
				default:
					return nil, err
				}
			}
		}
	}

	return l7Policy, nil
}

func checkL7PolicyAction(d *schema.ResourceData, action edgecloudV2.L7PolicyAction, redirectHTTPCode *int) error {
	redirectPoolID := d.Get(LBL7PolicyRedirectPoolIDField).(string)
	redirectURL := d.Get(LBL7PolicyRedirectURLField).(string)
	redirectPrefix := d.Get(LBL7PolicyRedirectPrefixField).(string)

	switch action {
	case edgecloudV2.L7PolicyActionRedirectPrefix:
		if redirectURL != "" || redirectPoolID != "" {
			return fmt.Errorf(
				"redirect_url and redirect_pool_id must be empty when action is set to %s", action)
		}
		if redirectPrefix == "" {
			return fmt.Errorf(
				"redirect_prefix must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionRedirectToPool:
		if redirectURL != "" || redirectPrefix != "" || redirectHTTPCode != nil {
			return fmt.Errorf(
				"redirect_url, redirect_prefix and redirect_http_code must be empty when action is set to %s", action)
		}
		if redirectPoolID == "" {
			return fmt.Errorf(
				"redirect_pool_id must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionRedirectToURL:
		if redirectPoolID != "" || redirectPrefix != "" {
			return fmt.Errorf(
				"redirect_prefix and redirect_pool_id must be empty when action is set to %s", action)
		}
		if redirectURL == "" {
			return fmt.Errorf(
				"redirect_url must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionReject:
		if redirectURL != "" || redirectPoolID != "" || redirectPrefix != "" || redirectHTTPCode != nil {
			return fmt.Errorf(
				"redirect_url, redirect_prefix, redirect_http_code and redirect_pool_id must be empty when action is set to %s", action)
		}
	}

	return nil
}

func CheckL7ListenerProtocol(ctx context.Context, client *edgecloudV2.Client, listenerID string) diag.Diagnostics {
	listener, _, err := client.Loadbalancers.ListenerGet(ctx, listenerID)
	if err != nil {
		return diag.Errorf("error from checking listener: %s", err)
	}
	if listener.Protocol != edgecloudV2.ListenerProtocolHTTP && listener.Protocol != edgecloudV2.ListenerProtocolTerminatedHTTPS {
		return diag.Errorf("%s protocol listeners do not support L7 policies", string(listener.Protocol))
	}
	return nil
}
