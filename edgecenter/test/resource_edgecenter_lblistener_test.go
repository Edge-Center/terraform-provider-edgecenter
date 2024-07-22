//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLBListener(t *testing.T) {
	//TODO: CLOUDDEV-862
	t.Skip("skipping test due to issue with IPv6 validation")

	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.LoadBalancersPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	lbFlavor := "lb1-1-2"
	opts := loadbalancers.CreateOpts{
		Name:   lbTestName,
		Flavor: &lbFlavor,
	}

	lbID, err := createTestLoadBalancerWithListener(client, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer loadbalancers.Delete(client, lbID)

	type Params struct {
		Name string
	}

	create := Params{"test"}

	update := Params{"test_new_name"}

	resourceName := "edgecenter_lblistener.acctest"

	ripTemplate := func(params *Params) string {
		return fmt.Sprintf(`
            resource "edgecenter_lblistener" "acctest" {
			  %s
              %s
			  name = "%s"
			  protocol = "TCP"
			  protocol_port = 36621
			  loadbalancer_id = "%s"
			}
		`, projectInfo(), regionInfo(), params.Name, lbID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: ripTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", create.Name),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBListenerDestroy,
		Steps: []resource.TestStep{
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

func testAccLBListenerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.LBListenersPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "edgecenter_lblistener" {
			_, err := listeners.Get(client, rs.Primary.ID).Extract()
			if err == nil {
				return fmt.Errorf("LBListener still exists")
			}
		}
	}

	return nil
}
