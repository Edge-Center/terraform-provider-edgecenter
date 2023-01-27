//go:build cloud
// +build cloud

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
)

func TestAccLoadBalancer(t *testing.T) {
	type Params struct {
		Name string
	}

	create := Params{"test"}

	update := Params{"test1"}

	fullName := "edgecenter_loadbalancerv2.acctest"

	ripTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_loadbalancerv2" "acctest" {
			  %s
              %s
			  name = "%s"
			  flavor = "lb1-1-2"
			}
		`, projectInfo(), regionInfo(), params.Name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: ripTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", create.Name),
				),
			},
			{
				Config: ripTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", update.Name),
				),
			},
		},
	})
}

func testAccLoadBalancerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	client, err := CreateTestClient(config.Provider, LoadBalancersPoint, versionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_loadbalancer" {
			continue
		}

		_, err := loadbalancers.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("LoadBalancer still exists")
		}
	}

	return nil
}
