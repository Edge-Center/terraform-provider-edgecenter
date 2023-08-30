//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/pools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccK8sPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	k8sClient, err := createTestClient(cfg.Provider, edgecenter.K8sPoint, edgecenter.VersionPointV1)
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
	defer deleteTestNetwork(netClient, networkID)

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
	defer keypairs.Delete(kpClient, keyPair.ID)

	nodeCountTestPtr := nodeCountTest
	dockerVolumeSizeTestPtr := dockerVolumeSizeTest
	maxNodeCountTestPtr := maxNodeCountTest
	k8sOpts := clusters.CreateOpts{
		Name:               clusterTestName,
		FixedNetwork:       networkID,
		FixedSubnet:        subnetID,
		AutoHealingEnabled: true,
		KeyPair:            keyPair.ID,
		Version:            clusterVersionTest,
		Pools: []pools.CreateOpts{{
			Name:             poolTestName,
			FlavorID:         flavorTest,
			NodeCount:        &nodeCountTestPtr,
			DockerVolumeSize: &dockerVolumeSizeTestPtr,
			DockerVolumeType: ockerVolumeTypeTest,
			MinNodeCount:     minNodeCountTest,
			MaxNodeCount:     &maxNodeCountTestPtr,
		}},
	}
	clusterID, err := createTestCluster(k8sClient, k8sOpts)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteTestCluster(k8sClient, clusterID)
	// we need to wait until upgrade will e finished
	time.Sleep(time.Second * 30)

	resourceName := "edgecenter_k8s_pool.acctest"
	type Params struct {
		Name             string
		Flavor           string
		MinNodeCount     int
		MaxNodeCount     int
		NodeCount        int
		DockerVolumeSize int
	}

	create := Params{
		Name:             "tf-pool1",
		Flavor:           "g1-standard-1-2",
		MinNodeCount:     1,
		MaxNodeCount:     1,
		NodeCount:        1,
		DockerVolumeSize: 2,
	}

	update := Params{
		Name:             "tf-pool2",
		Flavor:           "g1-standard-1-2",
		MinNodeCount:     1,
		MaxNodeCount:     2,
		NodeCount:        1,
		DockerVolumeSize: 2,
	}

	ipTemplate := func(p *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_k8s_pool" "acctest" {
			  %s
              %s
              cluster_id = "%s"
			  name = "%s"
			  flavor_id = "%s"
			  min_node_count = %d
			  max_node_count = %d
			  node_count = %d
			  docker_volume_size = %d
			}
		`, projectInfo(), regionInfo(), clusterID,
			p.Name, p.Flavor, p.MinNodeCount, p.MaxNodeCount,
			p.NodeCount, p.DockerVolumeSize)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccK8sPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", create.Name),
					resource.TestCheckResourceAttr(resourceName, "flavor_id", create.Flavor),
					resource.TestCheckResourceAttr(resourceName, "docker_volume_size", strconv.Itoa(create.DockerVolumeSize)),
					resource.TestCheckResourceAttr(resourceName, "min_node_count", strconv.Itoa(create.MinNodeCount)),
					resource.TestCheckResourceAttr(resourceName, "max_node_count", strconv.Itoa(create.MaxNodeCount)),
					resource.TestCheckResourceAttr(resourceName, "node_count", strconv.Itoa(create.NodeCount)),
				),
			},
			{
				Config: ipTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", update.Name),
					resource.TestCheckResourceAttr(resourceName, "flavor_id", update.Flavor),
					resource.TestCheckResourceAttr(resourceName, "docker_volume_size", strconv.Itoa(update.DockerVolumeSize)),
					resource.TestCheckResourceAttr(resourceName, "min_node_count", strconv.Itoa(update.MinNodeCount)),
					resource.TestCheckResourceAttr(resourceName, "max_node_count", strconv.Itoa(update.MaxNodeCount)),
					resource.TestCheckResourceAttr(resourceName, "node_count", strconv.Itoa(update.NodeCount)),
				),
			},
		},
	})
}

func testAccK8sPoolDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.K8sPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_k8s_pool" {
			continue
		}

		_, err := pools.Get(client, EC_CLUSTER_ID, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("k8s pool still exists")
		}
	}

	return nil
}
