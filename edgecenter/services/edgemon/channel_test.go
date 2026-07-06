//go:build test_edgemon

package edgemon_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/testacc"
)

var testProvider = testacc.NewProvider()

func TestAccEdgeMonChannel_basic(t *testing.T) {
	t.Parallel()
	name := testacc.UniqueName("tf-acc-edgemon-ch")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testacc.PreCheck(t) },
		ProviderFactories: testacc.Factory(testProvider),
		CheckDestroy:      testAccCheckChannelDestroy(testProvider),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("edgecenter_rmon_channel.test", "channel_name", name),
				),
			},
			{
				ResourceName:      "edgecenter_rmon_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccChannelConfig(name string) string {
	return fmt.Sprintf(`
resource "edgecenter_rmon_channel" "test" {
  receiver     = "telegram"
  token        = "tf-acc-token-%[1]s"
  channel_name = "%[1]s"
}
`, name)
}

func testAccCheckChannelDestroy(p *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cfg, ok := p.Meta().(*edgecenter.Config)
		if !ok || cfg == nil {
			return nil
		}
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "edgecenter_rmon_channel" {
				continue
			}
			id, err := strconv.Atoi(rs.Primary.ID)
			if err != nil {
				return err
			}
			receiver := rs.Primary.Attributes["receiver"]
			if _, err := cfg.RmonClient.Channel().Get(context.Background(), receiver, id); err == nil {
				return fmt.Errorf("rmon channel %d still exists", id)
			}
		}
		return nil
	}
}
