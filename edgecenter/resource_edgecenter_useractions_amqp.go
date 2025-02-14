package edgecenter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUserActionsSubscriptionAMQP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserActionsAMQPCreate,
		ReadContext:   resourceUserActionsAMQPRead,
		UpdateContext: resourceUserActionsAMQPUpdate,
		DeleteContext: resourceUserActionsAMQPDelete,
		Description:   `Resource provides access to user action logs and client subscription via AMQP.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			ConnectionStringField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A connection string of the following structure \"scheme://username:password@host:port/virtual_host\".",
			},
			ReceiveChildClientEventsField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true if you would like to receive user action logs of all clients with reseller_id matching the current client_id. Defaults to false.",
			},
			RoutingKeyField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Routing key.",
			},
			ExchangeAMQPField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Exchange name.",
			},
		},
	}
}

func resourceUserActionsAMQPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start creating AMQP subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	req := prepareAMQPSubscriptionCreateRequest(d)

	_, err = clientV2.UserActions.SubscribeAMQP(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceUserActionsAMQPRead(ctx, d, m)

	tflog.Debug(ctx, "Finished creating AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsAMQPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reading AMQP subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	subs, _, err := clientV2.UserActions.ListAMQPSubscriptions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if subs.Count == 0 {
		return nil
	}

	sub := subs.Results[0]

	d.SetId(strconv.Itoa(sub.ID))

	err = d.Set(ConnectionStringField, sub.ConnectionString)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(ReceiveChildClientEventsField, sub.ReceiveChildClientEvents)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(RoutingKeyField, sub.RoutingKey)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(ExchangeAMQPField, sub.Exchange)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reading AMQP subscription to the user actions")

	return nil
}

func resourceUserActionsAMQPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start updating AMQP subscription to the user actions")

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		rollbackAMQPSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	_, err = clientV2.UserActions.UnsubscribeAMQP(ctx)
	if err != nil {
		rollbackAMQPSubscriptionData(ctx, d)
		return diag.FromErr(err)
	}

	req := prepareAMQPSubscriptionCreateRequest(d)

	_, err = clientV2.UserActions.SubscribeAMQP(ctx, &req)
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

	clientV2, err := InitCloudClient(ctx, d, m, userActionsCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := clientV2.UserActions.UnsubscribeAMQP(ctx)
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
