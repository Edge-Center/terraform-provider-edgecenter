package useractions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func prepareAMQPSubscriptionCreateRequest(d *schema.ResourceData) edgecloudV2.AMQPSubscriptionCreateRequest {
	req := edgecloudV2.AMQPSubscriptionCreateRequest{
		ConnectionString:         d.Get(edgecenter.ConnectionStringField).(string),
		ReceiveChildClientEvents: d.Get(edgecenter.ReceiveChildClientEventsField).(bool),
	}

	if val, ok := d.GetOk(edgecenter.RoutingKeyField); ok {
		rk := val.(string)
		req.RoutingKey = &rk
	}

	if val, ok := d.GetOk(edgecenter.ExchangeAMQPField); ok {
		exchange := val.(string)
		req.Exchange = &exchange
	}

	return req
}

func rollbackAMQPSubscriptionData(ctx context.Context, d *schema.ResourceData) {
	cs, _ := d.GetChange(edgecenter.ConnectionStringField)
	err := d.Set(edgecenter.ConnectionStringField, cs)
	if err != nil {
		tflog.Error(ctx, "set old \"connection_string\" error: "+err.Error())
	}

	rce, _ := d.GetChange(edgecenter.ReceiveChildClientEventsField)
	err = d.Set(edgecenter.ReceiveChildClientEventsField, rce)
	if err != nil {
		tflog.Error(ctx, "set old \"receive_child_client_events\" error: "+err.Error())
	}

	rk, _ := d.GetChange(edgecenter.RoutingKeyField)
	err = d.Set(edgecenter.RoutingKeyField, rk)
	if err != nil {
		tflog.Error(ctx, "set old \"routing_key\" error: "+err.Error())
	}

	eAMQP, _ := d.GetChange(edgecenter.ExchangeAMQPField)
	err = d.Set(edgecenter.ExchangeAMQPField, eAMQP)
	if err != nil {
		tflog.Error(ctx, "set old \"exchange\" error: "+err.Error())
	}
}

func userActionsCloudClientConf() *edgecenter.CloudClientConf {
	return &edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
}

func prepareLogSubscriptionCreateRequest(d *schema.ResourceData) edgecloudV2.LogSubscriptionCreateRequest {
	req := edgecloudV2.LogSubscriptionCreateRequest{
		URL:             d.Get(edgecenter.SendUserActionLogsURLField).(string),
		AuthHeaderName:  d.Get(edgecenter.AuthHeaderNameField).(string),
		AuthHeaderValue: d.Get(edgecenter.AuthHeaderValueField).(string),
	}

	return req
}

func rollbackLogSubscriptionData(ctx context.Context, d *schema.ResourceData) {
	oldURL, _ := d.GetChange(edgecenter.SendUserActionLogsURLField)
	err := d.Set(edgecenter.SendUserActionLogsURLField, oldURL)
	if err != nil {
		tflog.Error(ctx, "set old \"url\" error: "+err.Error())
	}

	oldName, _ := d.GetChange(edgecenter.AuthHeaderNameField)
	err = d.Set(edgecenter.AuthHeaderNameField, oldName)
	if err != nil {
		tflog.Error(ctx, "set old \"auth_header_name\" error: "+err.Error())
	}

	oldValue, _ := d.GetChange(edgecenter.AuthHeaderValueField)
	err = d.Set(edgecenter.AuthHeaderValueField, oldValue)
	if err != nil {
		tflog.Error(ctx, "set old \"auth_header_value\" error: "+err.Error())
	}
}
