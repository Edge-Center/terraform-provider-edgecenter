package useractions

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const SubscriptionAMQPResource = "edgecenter_useractions_subscription_amqp"

func resourceUserActionsSubscriptionAMQP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserActionsAMQPCreate,
		ReadContext:   resourceUserActionsAMQPRead,
		UpdateContext: resourceUserActionsAMQPUpdate,
		DeleteContext: resourceUserActionsAMQPDelete,
		Description:   `Resource provides access to user action logs and client subscription via AMQP.`,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				clientID, err := strconv.Atoi(d.Id())
				if err != nil {
					return nil, fmt.Errorf("failed to parse client ID: %w", err)
				}
				if err = d.Set(edgecenter.ClientIDField, clientID); err != nil {
					return nil, fmt.Errorf("failed to set client ID: %w", err)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			edgecenter.ConnectionStringField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A connection string of the following structure \"scheme://username:password@host:port/virtual_host\".",
			},
			edgecenter.ReceiveChildClientEventsField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true if you would like to receive user action logs of all clients with reseller_id matching the current client_id. Defaults to false.",
			},
			edgecenter.RoutingKeyField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Routing key.",
			},
			edgecenter.ExchangeAMQPField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Exchange name.",
			},
			edgecenter.ClientIDField: {
				Type:        schema.TypeInt,
				Description: "The ID of the client.",
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}

func resourceUserActionsAMQPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start creating AMQP subscription to the user actions")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	req := prepareAMQPSubscriptionCreateRequest(d)
	opts := edgecloudV2.UserActionsOpts{ClientID: d.Get(edgecenter.ClientIDField).(int)}

	_, err = clientV2.UserActions.SubscribeAMQPWithOpts(ctx, &opts, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceUserActionsAMQPRead(ctx, d, m)

	tflog.Debug(ctx, "Finished creating AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsAMQPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reading AMQP subscription to the user actions")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	opts := edgecloudV2.UserActionsOpts{ClientID: d.Get(edgecenter.ClientIDField).(int)}
	subs, _, err := clientV2.UserActions.ListAMQPSubscriptionsWithOpts(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	switch {
	case subs.Count > 1:
		return diag.Errorf("forbidden to use admin token. Please use user token")
	case subs.Count == 0:
		d.SetId("")

		return nil
	}

	// A client can only have one subscription, so take the first element.
	sub := subs.Results[0]

	d.SetId(strconv.Itoa(sub.ID))

	err = d.Set(edgecenter.ConnectionStringField, sub.ConnectionString)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(edgecenter.ReceiveChildClientEventsField, sub.ReceiveChildClientEvents)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(edgecenter.RoutingKeyField, sub.RoutingKey)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(edgecenter.ExchangeAMQPField, sub.Exchange)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reading AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsAMQPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start updating AMQP subscription to the user actions")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		rollbackAMQPSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	opts := edgecloudV2.UserActionsOpts{ClientID: d.Get(edgecenter.ClientIDField).(int)}

	_, err = clientV2.UserActions.UnsubscribeAMQPWithOpts(ctx, &opts)
	if err != nil {
		rollbackAMQPSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	req := prepareAMQPSubscriptionCreateRequest(d)

	_, err = clientV2.UserActions.SubscribeAMQPWithOpts(ctx, &opts, &req)
	if err != nil {
		rollbackAMQPSubscriptionData(ctx, d)
		errCreate := resourceUserActionsAMQPCreate(ctx, d, m)
		if errCreate != nil {
			return diag.FromErr(err)
		}

		return diag.FromErr(err)
	}

	resourceUserActionsAMQPRead(ctx, d, m)

	tflog.Debug(ctx, "Finished updating AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsAMQPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start deleting AMQP subscription to the user actions")

	var diags diag.Diagnostics

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	opts := edgecloudV2.UserActionsOpts{ClientID: d.Get(edgecenter.ClientIDField).(int)}
	resp, err := clientV2.UserActions.UnsubscribeAMQPWithOpts(ctx, &opts)
	if err != nil {
		// If subscription for given client id does not exist, CloudAPI return 404
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.SetId("")

	tflog.Debug(ctx, "Finished deleting AMQP subscription to the user actions")

	return diags
}
