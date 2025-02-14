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

func TestAccEdgecenterUserActionsLogResource(t *testing.T) {
	t.Parallel()

	resourceName := "edgecenter_useractions_subscription_log.subs"

	checkAuthHeaderName := "Authorization"
	checkAuthHeaderValue := "Bearer eyJ0eXAiOawLr25Jh7Ix14"
	checkURL := "https://your-url.com/receive-user-action-messages"

	logSubTemplate := fmt.Sprintf(`
			resource "edgecenter_useractions_subscription_log" "subs" {
					auth_header_name = "%s"
					auth_header_value = "%s"
					url = "%s"
			}
		`, checkAuthHeaderName, checkAuthHeaderValue, checkURL)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLogSubsDestroy,
		Steps: []resource.TestStep{
			{
				Config: logSubTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.AuthHeaderNameField, checkAuthHeaderName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.AuthHeaderValueField, checkAuthHeaderValue),
					resource.TestCheckResourceAttr(resourceName, edgecenter.SendUserActionLogsURLField, checkURL),
				),
			},
		},
	})
}

func testAccLogSubsDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	clientV2, err := config.NewCloudClient()
	if err != nil {
		return err
	}

	resp, _, err := clientV2.UserActions.ListLogSubscriptions(context.Background())
	if err != nil {
		return fmt.Errorf("ListLogSubscriptions error: %w", err)
	}

	if resp.Count != 0 {
		return fmt.Errorf("log subscriptions still exists")
	}

	return nil
}
