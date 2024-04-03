//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLBL7RuleResource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	client := cfg.CloudClient

	t.Parallel()

	lbFlavor := "lb1-1-2"
	opts := edgecloudV2.LoadbalancerCreateRequest{
		Name:   "test-lb-l7rule",
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
		Name:           "test-l7rule",
		Action:         "REDIRECT_PREFIX",
		RedirectPrefix: "https://testsite.ru/",
	}

	t.Log("trying to create l7Policy...")
	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.L7Policies.Create, &l7CreateOpts, client, edgecenter.LBL7PolicyCreateTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("l7Policy is successfully created")

	l7PolicyID := taskResult.L7Polices[0]

	type Params struct {
		CompareType edgecloudV2.L7RuleCompareType
		Value       string
		Type        edgecloudV2.L7RuleType
	}

	create := Params{
		CompareType: edgecloudV2.L7RuleCompareTypeRegex,
		Value:       "/images*",
		Type:        edgecloudV2.L7RuleTypePath,
	}

	update := Params{
		CompareType: edgecloudV2.L7RuleCompareTypeRegex,
		Value:       "/images*",
		Type:        edgecloudV2.L7RuleTypePath,
	}
	resourceName := "edgecenter_lb_l7rule.l7rule_acctest"

	l7PolicyTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_lb_l7rule" "l7rule_acctest" {
			  %s
			  %s
			  type = "%s"
			  l7policy_id = "%s"
			  compare_type = "%s"
              value = "%s"
			  tags = ["a", "bbbb", "gg"]
			}
		`, projectInfo(), regionInfo(), params.Type, l7PolicyID, params.CompareType, params.Value)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBL7RuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: l7PolicyTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", string(create.Type)),
					resource.TestCheckResourceAttr(resourceName, "value", create.Value),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccLBL7RuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: l7PolicyTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", string(update.Type)),
					resource.TestCheckResourceAttr(resourceName, "value", update.Value),
				),
			},
		},
	})
}

func testAccLBL7RuleDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client := config.CloudClient
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "edgecenter_lb_l7rule" {
			_, _, err := client.L7Policies.Get(ctx, rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("l7Rule still exists")
			}
		}
	}

	return nil
}
