//go:build sweeper

package edgecenter_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func TestMain(m *testing.M) {
	registerSweepers()
	resource.TestMain(m)
}

func registerSweepers() {
	resource.AddTestSweepers("edgecenter_instance", &resource.Sweeper{
		Name: "edgecenter_instance",
		F:    sweepInstances,
	})
	resource.AddTestSweepers("edgecenter_loadbalancer", &resource.Sweeper{
		Name: "edgecenter_loadbalancer",
		F:    sweepLoadBalancers,
	})
	resource.AddTestSweepers("edgecenter_snapshot", &resource.Sweeper{
		Name:         "edgecenter_snapshot",
		F:            sweepSnapshots,
		Dependencies: []string{"edgecenter_instance"},
	})
	resource.AddTestSweepers("edgecenter_volume", &resource.Sweeper{
		Name:         "edgecenter_volume",
		F:            sweepVolumes,
		Dependencies: []string{"edgecenter_instance", "edgecenter_snapshot"},
	})
	resource.AddTestSweepers("edgecenter_subnet", &resource.Sweeper{
		Name:         "edgecenter_subnet",
		F:            sweepSubnets,
		Dependencies: []string{"edgecenter_instance", "edgecenter_loadbalancer"},
	})
	resource.AddTestSweepers("edgecenter_network", &resource.Sweeper{
		Name:         "edgecenter_network",
		F:            sweepNetworks,
		Dependencies: []string{"edgecenter_subnet"},
	})
	resource.AddTestSweepers("edgecenter_router", &resource.Sweeper{
		Name:         "edgecenter_router",
		F:            sweepRouters,
		Dependencies: []string{"edgecenter_subnet"},
	})
	resource.AddTestSweepers("edgecenter_securitygroup", &resource.Sweeper{
		Name: "edgecenter_securitygroup",
		F:    sweepSecurityGroups,
	})
	resource.AddTestSweepers("edgecenter_servergroup", &resource.Sweeper{
		Name: "edgecenter_servergroup",
		F:    sweepServerGroups,
	})
	resource.AddTestSweepers("edgecenter_keypair", &resource.Sweeper{
		Name:         "edgecenter_keypair",
		F:            sweepKeyPairs,
		Dependencies: []string{"edgecenter_instance"},
	})
	resource.AddTestSweepers("edgecenter_secret", &resource.Sweeper{
		Name: "edgecenter_secret",
		F:    sweepSecrets,
	})
	resource.AddTestSweepers("edgecenter_lifecyclepolicy", &resource.Sweeper{
		Name: "edgecenter_lifecyclepolicy",
		F:    sweepLifeCyclePolicies,
	})
	resource.AddTestSweepers("edgecenter_lb_l7policy", &resource.Sweeper{
		Name:         "edgecenter_lb_l7policy",
		F:            sweepL7Policies,
		Dependencies: []string{"edgecenter_loadbalancer"},
	})
}

func isTestResource(name string) bool {
	return strings.HasPrefix(name, testResourcePrefix+"-")
}

func waitForTask(ctx context.Context, client *edgecloudV2.Client, taskResp *edgecloudV2.TaskResponse) {
	if taskResp == nil || len(taskResp.Tasks) == 0 {
		return
	}
	utilV2.WaitForTaskComplete(ctx, client, taskResp.Tasks[0])
}

func sweepInstances(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	instances, _, err := client.Instances.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing instances: %w", err)
	}

	for _, inst := range instances {
		if !isTestResource(inst.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping instance: %s (%s)", inst.Name, inst.ID)
		taskResp, _, err := client.Instances.Delete(ctx, inst.ID, nil)
		if err != nil {
			log.Printf("[ERROR] Error deleting instance %s: %s", inst.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepLoadBalancers(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	lbs, _, err := client.Loadbalancers.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing loadbalancers: %w", err)
	}

	for _, lb := range lbs {
		if !isTestResource(lb.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping loadbalancer: %s (%s)", lb.Name, lb.ID)
		taskResp, _, err := client.Loadbalancers.Delete(ctx, lb.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting loadbalancer %s: %s", lb.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepSnapshots(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	snapshots, _, err := client.Snapshots.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing snapshots: %w", err)
	}

	for _, snap := range snapshots {
		if !isTestResource(snap.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping snapshot: %s (%s)", snap.Name, snap.ID)
		taskResp, _, err := client.Snapshots.Delete(ctx, snap.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting snapshot %s: %s", snap.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepVolumes(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	vols, _, err := client.Volumes.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing volumes: %w", err)
	}

	for _, vol := range vols {
		if !isTestResource(vol.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping volume: %s (%s)", vol.Name, vol.ID)
		taskResp, _, err := client.Volumes.Delete(ctx, vol.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting volume %s: %s", vol.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepSubnets(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	subnets, _, err := client.Subnetworks.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing subnets: %w", err)
	}

	for _, sub := range subnets {
		if !isTestResource(sub.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping subnet: %s (%s)", sub.Name, sub.ID)
		taskResp, _, err := client.Subnetworks.Delete(ctx, sub.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting subnet %s: %s", sub.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepNetworks(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	nets, _, err := client.Networks.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing networks: %w", err)
	}

	for _, net := range nets {
		if !isTestResource(net.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping network: %s (%s)", net.Name, net.ID)
		taskResp, _, err := client.Networks.Delete(ctx, net.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting network %s: %s", net.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepRouters(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	routers, _, err := client.Routers.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing routers: %w", err)
	}

	for _, r := range routers {
		if !isTestResource(r.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping router: %s (%s)", r.Name, r.ID)
		taskResp, _, err := client.Routers.Delete(ctx, r.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting router %s: %s", r.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepSecurityGroups(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	sgs, _, err := client.SecurityGroups.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing security groups: %w", err)
	}

	for _, sg := range sgs {
		if !isTestResource(sg.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping security group: %s (%s)", sg.Name, sg.ID)
		_, err := client.SecurityGroups.Delete(ctx, sg.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting security group %s: %s", sg.ID, err)
		}
	}

	return nil
}

func sweepServerGroups(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	sgs, _, err := client.ServerGroups.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing server groups: %w", err)
	}

	for _, sg := range sgs {
		if !isTestResource(sg.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping server group: %s (%s)", sg.Name, sg.ID)
		_, err := client.ServerGroups.Delete(ctx, sg.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting server group %s: %s", sg.ID, err)
		}
	}

	return nil
}

func sweepKeyPairs(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	kps, _, err := client.KeyPairs.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing keypairs: %w", err)
	}

	for _, kp := range kps {
		if !isTestResource(kp.SSHKeyName) {
			continue
		}
		log.Printf("[INFO] Sweeping keypair: %s (%s)", kp.SSHKeyName, kp.SSHKeyID)
		taskResp, _, err := client.KeyPairs.Delete(ctx, kp.SSHKeyID)
		if err != nil {
			log.Printf("[ERROR] Error deleting keypair %s: %s", kp.SSHKeyID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepSecrets(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	secs, _, err := client.Secrets.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}

	for _, sec := range secs {
		if !isTestResource(sec.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping secret: %s (%s)", sec.Name, sec.ID)
		taskResp, _, err := client.Secrets.Delete(ctx, sec.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting secret %s: %s", sec.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}

func sweepLifeCyclePolicies(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	policies, _, err := client.LifeCyclePolicies.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing lifecycle policies: %w", err)
	}

	for _, p := range policies {
		if !isTestResource(p.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping lifecycle policy: %s (%d)", p.Name, p.ID)
		_, err := client.LifeCyclePolicies.Delete(ctx, p.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting lifecycle policy %d: %s", p.ID, err)
		}
	}

	return nil
}

func sweepL7Policies(_ string) error {
	client, err := createTestCloudClient()
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	ctx := context.Background()

	policies, _, err := client.L7Policies.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing L7 policies: %w", err)
	}

	for _, p := range policies {
		if !isTestResource(p.Name) {
			continue
		}
		log.Printf("[INFO] Sweeping L7 policy: %s (%s)", p.Name, p.ID)
		taskResp, _, err := client.L7Policies.Delete(ctx, p.ID)
		if err != nil {
			log.Printf("[ERROR] Error deleting L7 policy %s: %s", p.ID, err)
			continue
		}
		waitForTask(ctx, client, taskResp)
	}

	return nil
}
