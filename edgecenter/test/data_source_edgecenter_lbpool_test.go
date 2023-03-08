//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/lbpools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccLBPoolDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
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

	clientPools, err := createTestClient(cfg.Provider, edgecenter.LBPoolsPoint, edgecenter.VersionPointV1)
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

	optsPool := lbpools.CreateOpts{
		Name:            poolTestName,
		Protocol:        types.ProtocolTypeHTTP,
		LoadBalancerID:  lbID,
		ListenerID:      listener.ID,
		LBPoolAlgorithm: types.LoadBalancerAlgorithmRoundRobin,
		HealthMonitor: &lbpools.CreateHealthMonitorOpts{
			Type:           types.HealthMonitorTypeHTTP,
			Delay:          5,
			MaxRetries:     10,
			Timeout:        10,
			MaxRetriesDown: 10,
			HTTPMethod:     types.HTTPMethodPointer(types.HTTPMethodGET),
			URLPath:        "/",
			ExpectedCodes:  "123,321",
		},
	}
	res, err := lbpools.Create(clientPools, optsPool).Extract()
	if err != nil {
		t.Fatal(err)
	}

	taskID := res.Tasks[0]
	lbPoolID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.LBPoolsCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		lbPoolID, err := lbpools.ExtractPoolIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve LBPool ID from task info: %w", err)
		}
		return lbPoolID, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	pool, err := lbpools.Get(clientPools, lbPoolID.(string)).Extract()
	if err != nil {
		t.Fatal(err)
	}

	resourceName := "data.edgecenter_lbpool.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_lbpool" "acctest" {
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
				Config: tpl(poolTestName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", poolTestName),
					resource.TestCheckResourceAttr(resourceName, "id", pool.ID),
				),
			},
		},
	})
}
