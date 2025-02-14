//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEdgecenterUserActionsAMQPDatasource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	checkConnectionCtring := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := true
	checkRoutingKey := "routing_key"

	amqpCreatReq := edgecloudV2.AMQPSubscriptionCreateRequest{
		ConnectionString:         checkConnectionCtring,
		ReceiveChildClientEvents: checkReceiveChildClientEvents,
		RoutingKey:               checkRoutingKey,
	}

	_, err = client.UserActions.SubscribeAMQP(ctx, &amqpCreatReq)
	if err != nil {
		t.Error(err)
	}

	defer client.UserActions.UnsubscribeAMQP(ctx)

	datasourceName := "data.edgecenter_useractions_subscription_amqp.subs"

	amqpSubTemplate := `
			data "edgecenter_useractions_subscription_amqp" "subs" {
			}
		`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: amqpSubTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ExchangeAMQPField, ""),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ReceiveChildClientEventsField, strconv.FormatBool(checkReceiveChildClientEvents)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.RoutingKeyField, checkRoutingKey),
				),
			},
		},
	})
}
