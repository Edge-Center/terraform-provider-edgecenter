//go:build cloud
// +build cloud

package edgecenter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
)

func TestAccBaremetal(t *testing.T) {
	if os.Getenv("LOCAL_TEST") == "" {
		t.Skip("skip test in ci")
	}

	fullName := "edgecenter_baremetal.acctest"

	ipTemplate := fmt.Sprintf(`
			resource "edgecenter_baremetal" "acctest" {
			  %s
              %s
			  name = "test sg"
			  flavor_id = "bm1-infrastructure-small"
			  image_id = "1ee7ccee-5003-48c9-8ae0-d96063af75b2"
			}
		`, projectInfo(), regionInfo())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccBaremetalDestroy,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", "test_sg"),
					resource.TestCheckResourceAttr(fullName, "flavor_id", "bm1-infrastructure-small"),
				),
			},
		},
	})
}

func testAccBaremetalDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	client, err := CreateTestClient(config.Provider, InstancePoint, versionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_baremetal" {
			continue
		}

		_, err := instances.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("baremetal instance %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
