//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSecurityGroup(t *testing.T) {
	t.Skip()
	t.Parallel()
	// TODO: https://tracker.yandex.ru/CLOUDDEV-473
	resourceName := "edgecenter_securitygroup.acctest"

	ipTemplate1 := fmt.Sprintf(`
			resource "edgecenter_securitygroup" "acctest" {
			  %s
              %s
			  name = "test"
			  metadata_map = {
				key1 = "val1"
				key2 = "val2"
			  }
			  security_group_rules {
			  	direction = "egress"
			    ethertype = "IPv4"
				protocol = "vrrp"
			  }
			}
		`, projectInfo(), regionInfo())

	ipTemplate2 := fmt.Sprintf(`
			resource "edgecenter_securitygroup" "acctest" {
			  %s
              %s
			  name = "test"
			  metadata_map = {
				key3 = "val3"
			  }
			  security_group_rules {
			  	direction = "egress"
			    ethertype = "IPv4"
				protocol = "vrrp"
			  }
			}
		`, projectInfo(), regionInfo())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata_map.key1", "val1"),
					resource.TestCheckResourceAttr(resourceName, "metadata_map.key2", "val2"),
					testAccCheckMetadata(t, resourceName, true, map[string]interface{}{
						"key1": "val1",
						"key2": "val2",
					}),
				),
			},
			{
				Config: ipTemplate2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata_map.key3", "val3"),
					testAccCheckMetadata(t, resourceName, true, map[string]interface{}{
						"key3": "val3",
					}),
					testAccCheckMetadata(t, resourceName, false, map[string]interface{}{
						"key1": "val1",
					}),
					testAccCheckMetadata(t, resourceName, false, map[string]interface{}{
						"key2": "val2",
					}),
				),
			},
		},
	})
}

func testAccSecurityGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.SecurityGroupPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_securitygroup" {
			continue
		}

		_, err := securitygroups.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("SecurityGroup still exists")
		}
	}

	return nil
}
