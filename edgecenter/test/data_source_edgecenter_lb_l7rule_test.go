//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func TestAccLBL7RuleDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	client := cfg.CloudClient

	t.Parallel()

	lbFlavor := "lb1-1-2"
	opts := edgecloudV2.LoadbalancerCreateRequest{
		Name:   "test-lb-l7rule-data-source",
		Flavor: lbFlavor,
		Listeners: []edgecloudV2.LoadbalancerListenerCreateRequest{{
			Name:         lbListenerTestName,
			ProtocolPort: 80,
			Protocol:     edgecloudV2.ListenerProtocolHTTP,
		}},
	}
	ctx := context.Background()

	t.Log("trying to create loadbalancer with listener...")
	lbID, err := createTestLoadBalancerWithListenerV2(ctx, client, opts)
	defer client.Loadbalancers.Delete(ctx, lbID)

	lb, _, err := client.Loadbalancers.Get(ctx, lbID)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("loadbalancer is successfully created")
	listener := lb.Listeners[0]

	l7CreateOpts := edgecloudV2.L7PolicyCreateRequest{
		ListenerID:     listener.ID,
		Name:           "test-l7rule",
		Action:         "REDIRECT_PREFIX",
		RedirectPrefix: "https://testsite.ru/",
	}

	t.Log("trying to create l7Policy...")
	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.L7Policies.Create, &l7CreateOpts, &client, edgecenter.LBL7PolicyCreateTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("l7Policy is successfully created")

	l7PolicyID := taskResult.L7Polices[0]

	l7RuleCreateReq := edgecloudV2.L7RuleCreateRequest{
		Tags:        nil,
		CompareType: "REGEX",
		Value:       "/images*",
		Type:        "PATH",
	}
	t.Log("trying to create l7rule...")
	result, _, err := client.L7Rules.Create(ctx, l7PolicyID, &l7RuleCreateReq)
	if err != nil {
		t.Fatal(err)
	}
	ruleTask, err := utilV2.WaitAndGetTaskInfo(ctx, &client, result.Tasks[0])
	if err != nil {
		t.Fatal(err)
	}
	ruleTaskResult, err := utilV2.ExtractTaskResultFromTask(ruleTask)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("l7Rule is successfully created")
	ruleID := ruleTaskResult.L7Rules[0]

	resourceName := "data.edgecenter_lb_l7rule.l7rule_acctest"

	tpl := func(id string) string {
		return fmt.Sprintf(`
			data "edgecenter_lb_l7rule" "l7rule_acctest" {
			  %s
              %s
              l7policy_id = "%s"
              id = "%s"
			}
		`, projectInfo(), regionInfo(), l7PolicyID, id)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(ruleID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", ruleID),
					resource.TestCheckResourceAttr(resourceName, "l7policy_id", l7PolicyID),
				),
			},
		},
	})
}
