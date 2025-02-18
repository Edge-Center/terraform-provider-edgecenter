package edgecenter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUserActionsListLogSubscriptions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserActionsLogRead,
		Description: `Data source provides access to user action logs and client subscription.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			SendUserActionLogsURLField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The URL to send user action logs for the specified client.",
			},
			AuthHeaderNameField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the authorization header.",
			},
			AuthHeaderValueField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The value of the authorization header",
			},
		},
	}
}

func dataSourceUserActionsLogRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reading log subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	subs, _, err := clientV2.UserActions.ListLogSubscriptions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if subs.Count == 0 {
		return diag.Errorf("log subscription to the user actions list is empty")
	}

	if subs.Count > 1 {
		return diag.FromErr(fmt.Errorf("forbidden to use admin token. Please use user token"))
	}

	sub := subs.Results[0]

	d.SetId(strconv.Itoa(sub.ID))

	err = d.Set(SendUserActionLogsURLField, sub.URL)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(AuthHeaderNameField, sub.AuthHeaderName)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(AuthHeaderValueField, sub.AuthHeaderValue)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reading log subscription to the user actions")

	return nil
}
