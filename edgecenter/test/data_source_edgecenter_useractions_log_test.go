//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEdgecenterUserActionsListLogSubscriptionsDatasource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	_, _ = client.UserActions.UnsubscribeLog(ctx)

	checkURL := "https://your-url.com/receive-user-action-messages"
	checkAuthHeaderValue := "Bearer eyJ0eXAiOawLr25Jh7Ix14"
	checkAuthHeaderName := "Authorization"

	logCreatReq := edgecloudV2.LogSubscriptionCreateRequest{
		URL:             checkURL,
		AuthHeaderValue: checkAuthHeaderValue,
		AuthHeaderName:  checkAuthHeaderName,
	}

	_, err = client.UserActions.SubscribeLog(ctx, &logCreatReq)
	if err != nil {
		t.Error(err)
	}

	defer client.UserActions.UnsubscribeLog(ctx)

	datasourceName := "data.edgecenter_useractions_subscription_log.subs"

	logSubTemplate := `
			data "edgecenter_useractions_subscription_log" "subs" {
			}
		`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: logSubTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.SendUserActionLogsURLField, checkURL),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.AuthHeaderNameField, checkAuthHeaderName),
				),
			},
		},
	})
}
