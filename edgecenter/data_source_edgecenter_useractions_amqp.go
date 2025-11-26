package edgecenter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceUserActionsListAMQPSubscriptions() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUserActionsAMQPRead,
		Description: `Data source provides access to user action logs and client subscription via AMQP.`,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			ConnectionStringField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A connection string of the following structure \"scheme://username:password@host:port/virtual_host\".",
			},
			ReceiveChildClientEventsField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Set to true if you would like to receive user action logs of all clients with reseller_id matching the current client_id. Defaults to false.",
			},
			RoutingKeyField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Routing key.",
			},
			ExchangeAMQPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Exchange name.",
			},
			ClientIDField: {
				Type:        schema.TypeInt,
				Description: "The ID of the client.",
				Optional:    true,
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

	opts := edgecloudV2.UserActionsOpts{ClientID: d.Get(ClientIDField).(int)}
	subs, _, err := clientV2.UserActions.ListAMQPSubscriptionsWithOpts(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	if subs.Count == 0 {
		return diag.Errorf("AMQP subscription to the user actions list is empty")
	}

	if subs.Count > 1 {
		return diag.FromErr(fmt.Errorf("forbidden to use admin token. Please use user token"))
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
