//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEdgecenterUserActionsListAMQPSubscriptionsDataSource(t *testing.T) {
	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	// Prepare a test environment by unsubscribing any existing subscriptions
	if _, err = client.UserActions.UnsubscribeAMQP(ctx); err != nil {
		t.Log(err)
	}

	checkConnectionString := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := true
	checkRoutingKey := "routing_key"

	amqpCreateReq := edgecloudV2.AMQPSubscriptionCreateRequest{
		ConnectionString:         checkConnectionString,
		ReceiveChildClientEvents: checkReceiveChildClientEvents,
		RoutingKey:               &checkRoutingKey,
	}

	_, err = client.UserActions.SubscribeAMQP(ctx, &amqpCreateReq)
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

func TestAccEdgecenterUserActions_ListAMQPSubscriptionsDataSource_WithClientID(t *testing.T) {
	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	if EC_CLIENT_ID == "" {
		t.Error("'EC_CLIENT_ID' must be set for acceptance test")
	}
	checkClientID, err := strconv.Atoi(EC_CLIENT_ID)
	if err != nil {
		t.Error(err)
	}

	checkConnectionString := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := true
	checkRoutingKey := "routing_key"

	opts := edgecloudV2.UserActionsOpts{ClientID: checkClientID}
	// Prepare a test environment by unsubscribing any existing subscriptions
	if _, err = client.UserActions.UnsubscribeAMQPWithOpts(ctx, &opts); err != nil {
		t.Log(err)
	}

	amqpCreateReq := edgecloudV2.AMQPSubscriptionCreateRequest{
		ConnectionString:         checkConnectionString,
		ReceiveChildClientEvents: checkReceiveChildClientEvents,
		RoutingKey:               &checkRoutingKey,
	}

	_, err = client.UserActions.SubscribeAMQPWithOpts(ctx, &opts, &amqpCreateReq)
	if err != nil {
		t.Error(err)
	}

	defer client.UserActions.UnsubscribeAMQPWithOpts(ctx, &opts)

	datasourceName := "data.edgecenter_useractions_subscription_amqp.subs"
	amqpSubTemplate := fmt.Sprintf(`
			data "edgecenter_useractions_subscription_amqp" "subs" {
				client_id = %d
			}
		`, checkClientID,
	)

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
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ConnectionStringField, checkConnectionString),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ClientIDField, EC_CLIENT_ID),
				),
			},
		},
	})
}
