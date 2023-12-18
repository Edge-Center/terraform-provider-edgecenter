//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccBaremetal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	resourceName := "edgecenter_baremetal.acctest"

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
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "test_sg"),
					resource.TestCheckResourceAttr(resourceName, "flavor_id", "bm1-infrastructure-small"),
				),
			},
		},
	})
}

func testAccBaremetalDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.InstancePoint, edgecenter.VersionPointV1)
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
