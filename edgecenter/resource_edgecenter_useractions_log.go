package edgecenter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUserActionsSubscriptionLog() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserActionsLogCreate,
		ReadContext:   resourceUserActionsLogRead,
		UpdateContext: resourceUserActionsLogUpdate,
		DeleteContext: resourceUserActionsLogDelete,
		Description:   `Resource provides access to user action logs and client subscription.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			SendUserActionLogsURLField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The URL to send user action logs for the specified client.",
			},
			AuthHeaderNameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the authorization header.",
			},
			AuthHeaderValueField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The value of the authorization header",
			},
		},
	}
}

func resourceUserActionsLogCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start creating log subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	req := prepareLogSubscriptionCreateRequest(d)

	_, err = clientV2.UserActions.SubscribeLog(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceUserActionsLogRead(ctx, d, m)

	tflog.Debug(ctx, "Finished creating log subscription to the user actions")

	return nil
}

func resourceUserActionsLogRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reading log subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	subs, _, err := clientV2.UserActions.ListLogSubscriptions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	switch {
	case subs.Count > 1:
		return diag.Errorf("forbidden to use admin token. Please use user token")
	case subs.Count == 0:
		if d.Id() != "" {
			return diag.Errorf(
				"current tfstate already has information about subscription, but subscription does not exist. " +
					"Please check the APIKey or clear the tfstate.")
		}

		return nil
	}

	// A client can only have one subscription, so take the first element.
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

func resourceUserActionsLogUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start updating log subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		rollbackLogSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	_, err = clientV2.UserActions.UnsubscribeLog(ctx)
	if err != nil {
		rollbackLogSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	req := prepareLogSubscriptionCreateRequest(d)

	_, err = clientV2.UserActions.SubscribeLog(ctx, &req)
	if err != nil {
		rollbackLogSubscriptionData(ctx, d)
		errCreate := resourceUserActionsLogCreate(ctx, d, m)
		if errCreate != nil {
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	resourceUserActionsLogRead(ctx, d, m)

	tflog.Debug(ctx, "Finished updating AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsLogDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start deleting log subscription to the user actions")

	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := clientV2.UserActions.UnsubscribeLog(ctx)
	if err != nil {
		// If subscription for given client id does not exist, CloudAPI return 404
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.SetId("")

	tflog.Debug(ctx, "Finished deleting log subscription to the user actions")

	return diags
}
