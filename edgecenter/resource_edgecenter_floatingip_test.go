//go:build cloud
// +build cloud

package edgecenter

import (
	"fmt"
	"testing"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/floatingip/v1/floatingips"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFloatingIP(t *testing.T) {
	fullName := "edgecenter_floatingip.acctest"

	ipTemplate := fmt.Sprintf(`
			resource "edgecenter_floatingip" "acctest" {
			  %s
              %s
			}
		`, projectInfo(), regionInfo())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccFloatingIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "fixed_ip_address", ""),
					resource.TestCheckResourceAttr(fullName, "port_id", ""),
				),
			},
		},
	})
}

func testAccFloatingIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	client, err := CreateTestClient(config.Provider, floatingIPsPoint, versionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_floatingip" {
			continue
		}

		_, err := floatingips.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("ReservedFixedIP still exists")
		}
	}

	return nil
}
