//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/floatingip/v1/floatingips"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccFloatingIP(t *testing.T) {
	t.Parallel()
	resourceName := "edgecenter_floatingip.acctest"

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
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "fixed_ip_address", ""),
					resource.TestCheckResourceAttr(resourceName, "port_id", ""),
				),
			},
		},
	})
}

func testAccFloatingIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.FloatingIPsPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_floatingip" {
			continue
		}

		_, err := floatingips.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("reservedFixedIP still exists")
		}
	}

	return nil
}
