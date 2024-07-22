//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLBL7PolicyDataSource(t *testing.T) {
	//TODO: CLOUDDEV-862
	t.Skip("skipping test due to issue with IPv6 validation")

	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	client := cfg.CloudClient

	t.Parallel()

	lbFlavor := "lb1-1-2"
	opts := edgecloudV2.LoadbalancerCreateRequest{
		Name:   "test-lb-l7policy-date-source",
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

	l7CreateOpts := edgecloudV2.L7PolicyCreateRequest{
		ListenerID:     listener.ID,
		Name:           "test-l7policy",
		Action:         "REDIRECT_PREFIX",
		RedirectPrefix: "https://test-prfix.ru/",
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.L7Policies.Create, &l7CreateOpts, client, edgecenter.LBL7PolicyCreateTimeout)
	if err != nil {
		t.Fatal(err)
	}

	l7PolicyID := taskResult.L7Polices[0]

	resourceName := "data.edgecenter_lb_l7policy.l7policy_acctest"

	tpl := func(id string) string {
		return fmt.Sprintf(`
			data "edgecenter_lb_l7policy" "l7policy_acctest" {
			  %s
              %s
              id = "%s"
			}
		`, projectInfo(), regionInfo(), id)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(l7PolicyID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", l7CreateOpts.Name),
					resource.TestCheckResourceAttr(resourceName, "id", l7PolicyID),
				),
			},
		},
	})
}
