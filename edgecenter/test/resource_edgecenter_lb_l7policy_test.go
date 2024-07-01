//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLBL7PolicyResource(t *testing.T) {
	//TODO: CLOUDDEV-862
	t.Skip("skipping test due to issue with IPv6 validation")

	client, err := createTestCloudClient()
	if err != nil {
		t.Fatal(err)
	}

	t.Parallel()

	lbFlavor := "lb1-1-2"
	opts := edgecloudV2.LoadbalancerCreateRequest{
		Name:   "test-lb-l7policy",
		Flavor: lbFlavor,
		Listeners: []edgecloudV2.LoadbalancerListenerCreateRequest{{
			Name:         lbListenerTestName,
			ProtocolPort: 80,
			Protocol:     edgecloudV2.ListenerProtocolHTTP,
		}},
	}
	ctx := context.Background()

	lbID, err := createTestLoadBalancerWithListenerV2(ctx, client, opts)
	defer client.Loadbalancers.Delete(ctx, lbID)

	lb, _, err := client.Loadbalancers.Get(ctx, lbID)
	if err != nil {
		t.Fatal(err)
	}
	listener := lb.Listeners[0]

	type Params struct {
		Name             string
		Action           string
		RedirectPrefix   string
		RedirectHTTPCode int
	}

	create := Params{
		Name:             "testL7policy",
		Action:           "REDIRECT_PREFIX",
		RedirectPrefix:   "https://accounts.edgecenter.online/",
		RedirectHTTPCode: 301,
	}

	update := Params{
		Name:             "testL7policy_updated",
		Action:           "REDIRECT_PREFIX",
		RedirectPrefix:   "https://accounts.edgecenter.ru/",
		RedirectHTTPCode: 302,
	}

	resourceName := "edgecenter_lb_l7policy.l7policy_acctest"

	l7PolicyTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_lb_l7policy" "l7policy_acctest" {
			  %s
			  %s
			  name = "%s"
			  action = "%s"
			  listener_id = "%s"
			  redirect_http_code = %d
			  redirect_prefix = "%s"
			  tags = ["a", "bbbb", "gg"]
			}
		`, projectInfo(), regionInfo(), params.Name, params.Action, listener.ID, params.RedirectHTTPCode, params.RedirectPrefix)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBL7PolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: l7PolicyTemplate(&create),
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
		CheckDestroy:      testAccLBL7PolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: l7PolicyTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", update.Name),
				),
			},
		},
	})
}

func testAccLBL7PolicyDestroy(s *terraform.State) error {
	client, err := createTestCloudClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "edgecenter_lb_l7policy" {
			_, _, err := client.L7Policies.Get(ctx, rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("l7Policy still exists")
			}
		}
	}

	return nil
}
