//go:build cloud_resource || cloud_reseller_resource

package edgecenter_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccEdgecenterUserActionsAMQPResource(t *testing.T) {
	prepareTestEnvironment(t)

	checkConnectionString := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := "true"
	checkRoutingKey := "routing_key"

	resourceName := "edgecenter_useractions_subscription_amqp.subs"

	amqpSubTemplate := fmt.Sprintf(`
			resource "edgecenter_useractions_subscription_amqp" "subs" {
					connection_string = "%s"
					receive_child_client_events = %s
					routing_key = "%s"
			}
		`, checkConnectionString, checkReceiveChildClientEvents, checkRoutingKey)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccAMQPSubsDestroy,
		Steps: []resource.TestStep{
			{
				Config: amqpSubTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ExchangeAMQPField, ""),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ReceiveChildClientEventsField, checkReceiveChildClientEvents),
					resource.TestCheckResourceAttr(resourceName, edgecenter.RoutingKeyField, checkRoutingKey),
				),
			},
		},
	})
}

func TestAccEdgecenterUserActionsAMQPResourceWithClientID(t *testing.T) {
	prepareTestEnvironment(t)

	checkConnectionString := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := "true"
	checkRoutingKey := "routing_key"

	resourceName := "edgecenter_useractions_subscription_amqp.subs"

	amqpSubTemplate := fmt.Sprintf(`
			resource "edgecenter_useractions_subscription_amqp" "subs" {
				connection_string = "%s"
				receive_child_client_events = %s
				routing_key = "%s"
				client_id = %s
		}
		`, checkConnectionString, checkReceiveChildClientEvents, checkRoutingKey, EC_CLIENT_ID,
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccAMQPSubsDestroyForClient,
		Steps: []resource.TestStep{
			{
				Config: amqpSubTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ExchangeAMQPField, ""),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ReceiveChildClientEventsField, checkReceiveChildClientEvents),
					resource.TestCheckResourceAttr(resourceName, edgecenter.RoutingKeyField, checkRoutingKey),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ClientIDField, EC_CLIENT_ID),
				),
			},
		},
	})
}

func prepareTestEnvironment(t *testing.T) {
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	if EC_CLIENT_ID == "" {
		t.Error("'EC_CLIENT_ID' must be set for acceptance test")
	}

	clientID, err := strconv.Atoi(EC_CLIENT_ID)
	if err != nil {
		t.Error(err)
	}

	opts := edgecloudV2.UserActionsOpts{ClientID: clientID}
	// Unsubscribing any existing subscriptions
	if _, err = client.UserActions.UnsubscribeAMQPWithOpts(context.Background(), &opts); err != nil {
		t.Log(err)
	}
}

func testAccAMQPSubsDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	clientV2, err := config.NewCloudClient()
	if err != nil {
		return err
	}

	resp, _, err := clientV2.UserActions.ListAMQPSubscriptions(context.Background())
	if err != nil {
		return fmt.Errorf("ListAMQPSubscriptions error: %w", err)
	}

	if resp.Count != 0 {
		return fmt.Errorf("AMQP subscriptions still exists")
	}

	return nil
}

func testAccAMQPSubsDestroyForClient(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	clientV2, err := config.NewCloudClient()
	if err != nil {
		return err
	}

	clientID, err := strconv.Atoi(EC_CLIENT_ID)
	if err != nil {
		return err
	}

	opts := edgecloudV2.UserActionsOpts{ClientID: clientID}
	if opts.ClientID == 0 {
		return fmt.Errorf("client id is not set")
	}
	resp, _, err := clientV2.UserActions.ListAMQPSubscriptionsWithOpts(context.Background(), &opts)
	if err != nil {
		return fmt.Errorf("ListAMQPSubscriptions error: %w", err)
	}

	if resp.Count != 0 {
		return fmt.Errorf("AMQP subscriptions still exists")
	}

	return nil
}
