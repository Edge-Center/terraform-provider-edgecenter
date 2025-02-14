package edgecenter

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func prepareAMQPSubscriptionCreateRequest(d *schema.ResourceData) edgecloudV2.AMQPSubscriptionCreateRequest {
	req := edgecloudV2.AMQPSubscriptionCreateRequest{
		ConnectionString:         d.Get(ConnectionStringField).(string),
		ReceiveChildClientEvents: d.Get(ReceiveChildClientEventsField).(bool),
	}

	if val, ok := d.GetOk(RoutingKeyField); ok {
		req.RoutingKey = val.(string)
	}

	if val, ok := d.GetOk(ExchangeAMQPField); ok {
		req.Exchange = val.(string)
	}

	return req
}

func rollbackAMQPSubscriptionData(ctx context.Context, d *schema.ResourceData) {
	cs, _ := d.GetChange(ConnectionStringField)
	err := d.Set(ConnectionStringField, cs)
	if err != nil {
		tflog.Error(ctx, "set old \"connection_string\" error: "+err.Error())
	}

	rce, _ := d.GetChange(ReceiveChildClientEventsField)
	err = d.Set(ReceiveChildClientEventsField, rce)
	if err != nil {
		tflog.Error(ctx, "set old \"receive_child_client_events\" error: "+err.Error())
	}

	rk, _ := d.GetChange(RoutingKeyField)
	err = d.Set(RoutingKeyField, rk)
	if err != nil {
		tflog.Error(ctx, "set old \"routing_key\" error: "+err.Error())
	}

	eAMQP, _ := d.GetChange(ExchangeAMQPField)
	err = d.Set(ExchangeAMQPField, eAMQP)
	if err != nil {
		tflog.Error(ctx, "set old \"exchange\" error: "+err.Error())
	}
}

func userActionsCloudClientConf() *CloudClientConf {
	return &CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
}

func prepareLogSubscriptionCreateRequest(d *schema.ResourceData) edgecloudV2.LogSubscriptionCreateRequest {
	req := edgecloudV2.LogSubscriptionCreateRequest{
		URL:             d.Get(SendUserActionLogsURLField).(string),
		AuthHeaderName:  d.Get(AuthHeaderNameField).(string),
		AuthHeaderValue: d.Get(AuthHeaderValueField).(string),
	}

	return req
}

func rollbackLogSubscriptionData(ctx context.Context, d *schema.ResourceData) {
	oldURL, _ := d.GetChange(SendUserActionLogsURLField)
	err := d.Set(SendUserActionLogsURLField, oldURL)
	if err != nil {
		tflog.Error(ctx, "set old \"url\" error: "+err.Error())
	}

	oldName, _ := d.GetChange(AuthHeaderNameField)
	err = d.Set(AuthHeaderNameField, oldName)
	if err != nil {
		tflog.Error(ctx, "set old \"auth_header_name\" error: "+err.Error())
	}

	oldValue, _ := d.GetChange(AuthHeaderValueField)
	err = d.Set(AuthHeaderValueField, oldValue)
	if err != nil {
		tflog.Error(ctx, "set old \"auth_header_value\" error: "+err.Error())
	}
}
