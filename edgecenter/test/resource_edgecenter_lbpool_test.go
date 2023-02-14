//go:build cloud

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/lbpools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLBPool(t *testing.T) {
	t.Skip()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, edgecenter.LoadBalancersPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientListener, err := CreateTestClient(cfg.Provider, edgecenter.LBListenersPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := loadbalancers.CreateOpts{
		Name: lbTestName,
		Listeners: []loadbalancers.CreateListenerOpts{{
			Name:         lbListenerTestName,
			ProtocolPort: 80,
			Protocol:     types.ProtocolTypeHTTP,
		}},
	}

	lbID, err := createTestLoadBalancerWithListener(client, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer loadbalancers.Delete(client, lbID)

	ls, err := listeners.ListAll(clientListener, listeners.ListOpts{LoadBalancerID: &lbID})
	if err != nil {
		t.Fatal(err)
	}
	listener := ls[0]

	type Params struct {
		Name        string
		LBAlgorithm string
	}

	create := Params{"test", "ROUND_ROBIN"}

	update := Params{"test1", "LEAST_CONNECTIONS"}

	fullName := "edgecenter_lbpool.acctest"

	ripTemplate := func(params *Params) string {
		return fmt.Sprintf(`
            resource "edgecenter_lbpool" "acctest" {
			  %s
              %s
			  name = "%s"
			  protocol = "HTTP"
			  lb_algorithm = "%s"
			  loadbalancer_id = "%s"
			  listener_id = "%s"
			}
		`, projectInfo(), regionInfo(), params.Name, params.LBAlgorithm, lbID, listener.ID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: ripTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", create.Name),
					resource.TestCheckResourceAttr(fullName, "lb_algorithm", create.LBAlgorithm),
				),
			},
			{
				Config: ripTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", update.Name),
					resource.TestCheckResourceAttr(fullName, "lb_algorithm", update.LBAlgorithm),
				),
			},
		},
	})
}

func testAccLBPoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := CreateTestClient(config.Provider, edgecenter.LBPoolsPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "edgecenter_lbpool" {
			_, err := lbpools.Get(client, rs.Primary.ID).Extract()
			if err == nil {
				return fmt.Errorf("LBPool still exists")
			}
		}
	}

	return nil
}
