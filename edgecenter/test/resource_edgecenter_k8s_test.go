//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccK8s(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	netClient, err := createTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	subnetClient, err := createTestClient(cfg.Provider, edgecenter.SubnetPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	kpClient, err := createTestClient(cfg.Provider, edgecenter.KeypairsPoint, edgecenter.VersionPointV2)
	if err != nil {
		t.Fatal(err)
	}

	netOpts := networks.CreateOpts{
		Name:         networkTestName,
		CreateRouter: true,
	}
	networkID, err := createTestNetwork(netClient, netOpts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { deleteTestNetwork(netClient, networkID) })

	gw := net.ParseIP("")
	subnetOpts := subnets.CreateOpts{
		Name:                   subnetTestName,
		NetworkID:              networkID,
		ConnectToNetworkRouter: true,
		EnableDHCP:             true,
		GatewayIP:              &gw,
	}

	subnetID, err := createTestSubnet(subnetClient, subnetOpts)
	if err != nil {
		t.Fatal(err)
	}

	// update our new network router so that the k8s nodes will have access to the Nexus
	// registry to download images
	if err := patchRouterForK8S(cfg.Provider, networkID); err != nil {
		t.Fatal(err)
	}

	pid, err := strconv.Atoi(os.Getenv("TEST_PROJECT_ID"))
	if err != nil {
		t.Fatal(err)
	}

	kpOpts := keypairs.CreateOpts{
		Name:      kpTestName,
		PublicKey: pkTest,
		ProjectID: pid,
	}
	keyPair, err := keypairs.Create(kpClient, kpOpts).Extract()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { keypairs.Delete(kpClient, keyPair.ID) })

	resourceName := "edgecenter_k8s.acctest"
	k8sName := testName("k8s")
	k8sPoolName := testName("k8s-pool")

	ipTemplate := fmt.Sprintf(`
			resource "edgecenter_k8s" "acctest" {
			  %s
              %s
              name = "%s"
			  fixed_network = "%s"
			  fixed_subnet = "%s"
              keypair = "%s"
			  pool {
				name = "%s"
				flavor_id = "g1-standard-1-2"
				min_node_count = 1
				max_node_count = 1
				node_count = 1
				docker_volume_size = 2
			  }

			}
		`, projectInfo(), regionInfo(), k8sName, networkID, subnetID, keyPair.ID, k8sPoolName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccK8sDestroy,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", k8sName),
				),
			},
		},
	})
}

func testAccK8sDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.K8sPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_k8s" {
			continue
		}

		_, err := clusters.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("k8s cluster still exists")
		}
	}

	return nil
}
