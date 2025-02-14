package edgecenter

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataUserActionsSubscriptionAMQP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserActionsAMQPRead,
		Description: `Data source provides access to user action logs and client subscription via AMQP.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			ConnectionStringField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
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

func dataSourceUserActionsAMQPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diag.Errorf("AMQP subscription to the user actions list is empty")
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
