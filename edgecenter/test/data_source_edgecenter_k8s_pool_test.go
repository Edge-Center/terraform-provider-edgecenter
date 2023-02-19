//go:build cloud

package edgecenter_test

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/pools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccK8sPoolDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	k8sClient, err := CreateTestClient(cfg.Provider, edgecenter.K8sPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	netClient, err := CreateTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	subnetClient, err := CreateTestClient(cfg.Provider, edgecenter.SubnetPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	kpClient, err := CreateTestClient(cfg.Provider, edgecenter.KeypairsPoint, edgecenter.VersionPointV2)
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

	subnetID, err := CreateTestSubnet(subnetClient, subnetOpts)
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
		Name:      kpName,
		PublicKey: pkTest,
		ProjectID: pid,
	}
	keyPair, err := keypairs.Create(kpClient, kpOpts).Extract()
	if err != nil {
		t.Fatal(err)
	}
	defer keypairs.Delete(kpClient, keyPair.ID)

	k8sOpts := clusters.CreateOpts{
		Name:               testClusterName,
		FixedNetwork:       networkID,
		FixedSubnet:        subnetID,
		AutoHealingEnabled: true,
		KeyPair:            keyPair.ID,
		Version:            testClusterVersion,
		Pools: []pools.CreateOpts{{
			Name:             testClusterPoolName,
			FlavorID:         testPoolFlavor,
			NodeCount:        testNodeCount,
			DockerVolumeSize: testDockerVolumeSize,
			DockerVolumeType: testDockerVolumeType,
			MinNodeCount:     testMinNodeCount,
			MaxNodeCount:     testMaxNodeCount,
		}},
	}
	clusterID, err := createTestCluster(k8sClient, k8sOpts)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteTestCluster(k8sClient, clusterID)

	cluster, err := clusters.Get(k8sClient, clusterID).Extract()
	if err != nil {
		t.Fatal(err)
	}
	pool := cluster.Pools[0]

	fullName := "data.edgecenter_k8s_pool.acctest"
	ipTemplate := fmt.Sprintf(`
			data "edgecenter_k8s_pool" "acctest" {
			  %s
              %s
              cluster_id = "%s"
			  pool_id = "%s"
			}
		`, projectInfo(), regionInfo(), clusterID, pool.UUID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: ipTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "cluster_id", clusterID),
					resource.TestCheckResourceAttr(fullName, "pool_id", pool.UUID),
					resource.TestCheckResourceAttr(fullName, "name", pool.Name),
				),
			},
		},
	})
}
