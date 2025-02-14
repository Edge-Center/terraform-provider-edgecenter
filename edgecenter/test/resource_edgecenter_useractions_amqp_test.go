//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccEdgecenterUserActionsAMQPResource(t *testing.T) {
	t.Parallel()

	checkConnectionCtring := "amqps://guest:guest@192.168.123.21:5671/user_action_events"
	checkReceiveChildClientEvents := "true"
	checkRoutingKey := "routing_key"

	resourceName := "edgecenter_useractions_subscription_amqp.subs"

	amqpSubTemplate := fmt.Sprintf(`
			resource "edgecenter_useractions_subscription_amqp" "subs" {
					connection_string = "%s"
					receive_child_client_events = %s
					routing_key = "%s"
			}
		`, checkConnectionCtring, checkReceiveChildClientEvents, checkRoutingKey)

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
