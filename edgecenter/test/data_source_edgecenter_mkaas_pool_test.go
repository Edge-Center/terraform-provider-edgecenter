//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccDataSourceMKaaSPool(t *testing.T) {
	if os.Getenv("RUN_MKAAS_IT") != "1" {
		t.Skip("This test requires RUN_MKAAS_IT=1")
	}

	ctx := context.Background()

	// --- env
	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	cloudAPIURL := requireEnv(t, "EC_API")
	projectID := requireEnv(t, "TEST_PROJECT_ID")
	regionID := requireEnv(t, "TEST_MKAAS_REGION_ID")

	cpFlavor := os.Getenv("EC_MKAAS_CP_FLAVOR")
	if cpFlavor == "" {
		cpFlavor = MKaaSCpFlavor
	}

	volType := os.Getenv("EC_MKAAS_VOLUME_TYPE")
	if volType == "" {
		volType = MKaaSVolumeType
	}

	k8sVersion := os.Getenv("EC_MKAAS_K8S_VERSION")
	if k8sVersion == "" {
		k8sVersion = MKaaSK8sVersion
	}

	base := "tf-mkaas-ds-test-" + strings.ToLower(randomSuffix())

	client, err := CreateClient(t, token, cloudAPIURL, projectID, regionID)
	if err != nil {
		t.Fatalf("failed to create V2 client: %v", err)
	}

	// --- keypair
	keypairName := base + "-key"
	keypairID, err := CreateTestKeypair(t, client, keypairName)
	if err != nil {
		t.Fatalf("create keypair: %v", err)
	}
	defer func() {
		if err := DeleteTestKeypair(t, client, keypairID); err != nil {
			t.Logf("cleanup keypair: %v", err)
		}
	}()

	// --- network
	networkName := base + "-net"
	networkID, err := CreateTestNetwork(client, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	defer func() {
		if err := DeleteTestNetwork(client, networkID); err != nil {
			t.Logf("cleanup network: %v", err)
		}
	}()

	// --- subnet
	subnetName := base + "-subnet"
	subnetID, err := CreateTestSubnet(client, &edgecloudV2.SubnetworkCreateRequest{
		Name:                   subnetName,
		NetworkID:              networkID,
		CIDR:                   "192.168.123.0/24",
		EnableDHCP:             true,
		ConnectToNetworkRouter: true,
	})
	if err != nil {
		t.Fatalf("create subnet: %v", err)
	}
	defer func() {
		if err := DeleteTestSubnet(client, subnetID); err != nil {
			t.Logf("cleanup subnet: %v", err)
		}
	}()

	// --- cluster
	clusterName := base + "-cls"
	clusterTask, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.MkaaS.ClusterCreate, edgecloudV2.MkaaSClusterCreateRequest{
		Name:           clusterName,
		SSHKeyPairName: keypairName,
		NetworkID:      networkID,
		SubnetID:       subnetID,
		ControlPlane: edgecloudV2.ControlPlaneCreateRequest{
			Flavor:     cpFlavor,
			NodeCount:  1,
			VolumeSize: 30,
			VolumeType: edgecloudV2.VolumeType(volType),
			Version:    k8sVersion,
		},
	}, client, edgecenter.MKaaSClusterCreateTimeout)
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	if len(clusterTask.MkaasClusters) == 0 {
		t.Fatalf("cluster id is empty")
	}
	clusterIDFloat := clusterTask.MkaasClusters[0]
	clusterID := fmt.Sprintf("%.0f", clusterIDFloat)
	defer func() {
		if err := DeleteTestMKaaSCluster(t, client, clusterID); err != nil {
			t.Logf("cleanup cluster: %v", err)
		}
	}()

	// --- pool
	poolName := base + "-pool"
	createPoolResp, _, err := client.MkaaS.PoolCreate(ctx, int(clusterIDFloat), edgecloudV2.MkaaSPoolCreateRequest{
		Name:       poolName,
		Flavor:     cpFlavor,
		NodeCount:  1,
		VolumeSize: 30,
		VolumeType: edgecloudV2.VolumeType(volType),
	})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	taskID := createPoolResp.Tasks[0]
	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID, edgecenter.MKaaSPoolCreateTimeout)
	if err != nil {
		t.Fatalf("wait pool task: %v", err)
	}
	taskResult, err := utilV2.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		t.Fatalf("extract pool task result: %v", err)
	}
	if len(taskResult.MkaasPools) == 0 {
		t.Fatalf("pool id is empty")
	}
	poolID := fmt.Sprintf("%.0f", taskResult.MkaasPools[0])

	resourceName := "data.edgecenter_mkaas_pool.acctest"
	cfg := fmt.Sprintf(`
		data "edgecenter_mkaas_pool" "acctest" {
		  project_id = %s
          region_id  = %s
		  cluster_id = %s
		  pool_id    = %s
		}
	`, projectID, regionID, clusterID, poolID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", poolID),
					resource.TestCheckResourceAttr(resourceName, "cluster_id", clusterID),
					resource.TestCheckResourceAttr(resourceName, "name", poolName),
				),
			},
		},
	})
}

// randomSuffix returns a short pseudo-random suffix to avoid name collisions.
func randomSuffix() string {
	return strings.ToLower(random.UniqueId())
}
