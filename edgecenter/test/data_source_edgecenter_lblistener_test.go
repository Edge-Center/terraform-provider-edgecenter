//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLBListenerDataSource(t *testing.T) {
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

	clientListener, err := createTestClient(cfg.Provider, edgecenter.LBListenersPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := loadbalancers.CreateOpts{
		Name: lbTestName,
		Listeners: []loadbalancers.CreateListenerOpts{{
			Name:         lbListenerTestName,
			ProtocolPort: 80,
			Protocol:     types.ProtocolTypeHTTP,
			AllowedCIDRs: []string{"127.0.0.0/24"},
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

	resourceName := "data.edgecenter_lblistener.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_lblistener" "acctest" {
			  %s
              %s
              name = "%s"
			}
		`, projectInfo(), regionInfo(), name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(lbListenerTestName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", lbListenerTestName),
					resource.TestCheckResourceAttr(resourceName, "id", listener.ID),
					resource.TestCheckResourceAttr(resourceName, "allowed_cidrs.#", strconv.Itoa(len(listener.AllowedCIDRs))),
				),
			},
		},
	})
}
