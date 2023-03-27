//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLoadBalancer(t *testing.T) {
	type Params struct {
		Name string
	}

	create := Params{"test"}

	update := Params{"test1"}

	resourceName := "edgecenter_loadbalancerv2.acctest"

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
				Config: fmt.Sprintf(`
resource "edgecenter_loadbalancerv2" "acctest" {
	%s
	%s
	name = "%s"
	flavor = "lb1-1-2"
	vip_port_id = "%s"
	vip_network_id = "%s"
}`, projectInfo(), regionInfo(), create.Name, "vip_port_id_123", "vip_network_id_123"),
				ExpectError: regexp.MustCompile("Conflicting configuration arguments"),
				PlanOnly:    true,
				Destroy:     false,
			},
			{
				Config: ripTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", create.Name),
				),
			},
			{
				Config: ripTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", update.Name),
				),
			},
		},
	})
}

func testAccLoadBalancerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.LoadBalancersPoint, edgecenter.VersionPointV1)
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
