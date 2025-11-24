package edgecenter_test

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	MKaaSVolumeType = "ssd_hiiops"
	MKaaSK8sVersion = "v1.31.0"
	MKaaSCpFlavor   = "g3-standard-2-4"
)

func TestMKaaSCluster_ApplyUpdateImportDestroy(t *testing.T) {
	if os.Getenv("RUN_MKAAS_IT") != "1" {
		t.Skip("This test requires RUN_MKAAS_IT=1")
	}

	t.Log("Starting TestMKaaSCluster_ApplyUpdateImportDestroy")

	// --- env
	t.Log("Reading environment variables...")
	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	endpoint := os.Getenv("EC_API")
	if endpoint == "" {
		endpoint = "https://api.edgecenter.ru"
	}
	t.Logf("Using endpoint: %s", endpoint)
	projectID := requireEnv(t, "TEST_PROJECT_ID")
	regionID := requireEnv(t, "TEST_MKAAS_REGION_ID")

	cpFlavor := os.Getenv("EC_MKAAS_CP_FLAVOR")
	if cpFlavor == "" {
		cpFlavor = MKaaSCpFlavor
	}

	cpVolumeType := os.Getenv("EC_MKAAS_VOLUME_TYPE")
	if cpVolumeType == "" {
		cpVolumeType = MKaaSVolumeType
	}

	cpVersion := os.Getenv("EC_MKAAS_K8S_VERSION")
	if cpVersion == "" {
		cpVersion = MKaaSK8sVersion
	}

	t.Log("Creating  client...")
	var err error
	client, err := CreateClient(t, token, endpoint, projectID, regionID)
	require.NoError(t, err, "failed to create client")

	baseName := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	keypairName := baseName + "-key"
	t.Logf("Creating SSH keypair with name: %s", keypairName)
	keypairID, err := CreateTestKeypair(t, client, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Logf("SSH keypair created successfully with ID: %s, name: %s", keypairID, keypairName)
	t.Cleanup(func() {
		if err := DeleteTestKeypair(t, client, keypairID); err != nil {
			t.Errorf("cleanup failed: delete SSH keypair %s: %v", keypairID, err)
		}
	})

	t.Log("Creating network...")
	networkName := baseName + "-net"
	t.Logf("Creating network with name: %s", networkName)
	networkID, err := CreateTestNetwork(client, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	require.NoError(t, err, "failed to create network")
	t.Logf("Network created successfully with ID: %s", networkID)
	t.Cleanup(func() {
		if err := DeleteTestNetwork(client, networkID); err != nil {
			t.Errorf("cleanup failed: delete network %s: %v", networkID, err)
		}
	})

	t.Log("Creating subnet...")
	subnetName := baseName + "-subnet"
	t.Logf("Creating subnet with name: %s in network: %s", subnetName, networkID)
	ip := net.ParseIP("192.168.42.1")
	subnetID, err := CreateTestSubnet(client, &edgecloudV2.SubnetworkCreateRequest{
		Name:                   subnetName,
		NetworkID:              networkID,
		CIDR:                   "192.168.42.0/24",
		EnableDHCP:             true,
		ConnectToNetworkRouter: true,
		GatewayIP:              &ip,
	})
	require.NoError(t, err, "failed to create subnet")
	t.Logf("Subnet created successfully with ID: %s", subnetID)
	t.Cleanup(func() {
		if err := DeleteTestSubnet(client, subnetID); err != nil {
			t.Errorf("cleanup failed: delete subnet %s: %v", subnetID, err)
		}
	})

	nameV1 := baseName + "-v1"
	nameV2 := baseName + "-v2"

	data := tfData{
		Token:        token,
		Endpoint:     endpoint,
		ProjectID:    projectID,
		RegionID:     regionID,
		NetworkID:    networkID,
		SubnetID:     subnetID,
		SSHKeypair:   keypairName,
		Name:         nameV1,
		CPFlavor:     cpFlavor,
		CPNodeCount:  1,
		CPVolumeSize: 30,
		CPVolumeType: cpVolumeType,
		CPVersion:    cpVersion,
	}

	// --- CREATE cluster
	cluster, err := CreateCluster(t, data)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	var testSucceed bool
	t.Cleanup(func() {
		if cluster != nil && !testSucceed {
			if err := DeleteTestMKaaSCluster(t, client, cluster.ID); err != nil {
				t.Errorf("cleanup failed: delete cluster %s via API: %v", cluster.ID, err)
			}
		}
	})
	// Check cluster
	require.Equalf(t, cluster.ID, output(t, cluster, "cluster_id"), "%s mismatch", "cluster_id non-empty")
	require.Equalf(t, nameV1, output(t, cluster, "cluster_name"), "%s mismatch", "cluster_name")
	require.Equalf(t, projectID, output(t, cluster, "out_project_id"), "%s mismatch", "project_id")
	require.Equalf(t, regionID, output(t, cluster, "out_region_id"), "%s mismatch", "region_id")
	require.Equalf(t, keypairName, output(t, cluster, "out_ssh_keypair_name"), "%s mismatch", "ssh_keypair_name")
	require.Equalf(t, networkID, output(t, cluster, "out_network_id"), "%s mismatch", "network_id")
	require.Equalf(t, subnetID, output(t, cluster, "out_subnet_id"), "%s mismatch", "subnet_id")
	require.Equalf(t, cpFlavor, output(t, cluster, "out_cp_flavor"), "%s mismatch", "control_plane.flavor")
	require.Equalf(t, "1", output(t, cluster, "out_cp_node_count"), "%s mismatch", "control_plane.node_count")
	require.Equalf(t, "30", output(t, cluster, "out_cp_volume_size"), "%s mismatch", "control_plane.volume_size")
	require.Equalf(t, cpVolumeType, output(t, cluster, "out_cp_volume_type"), "%s mismatch", "control_plane.volume_type")
	require.Equalf(t, cpVersion, output(t, cluster, "out_k8s_version"), "%s mismatch", "control_plane.version")

	// --- UPDATE cluster
	err = cluster.UpdateCluster(t, func(d *tfData) {
		d.Name = nameV2
		d.CPNodeCount = 3
	})

	if err != nil {
		t.Fatalf("failed to update cluster: %v", err)
	}
	require.NoError(t, err, "failed to update cluster")
	require.Equalf(t, "3", output(t, cluster, "out_cp_node_count"), "%s mismatch", "control_plane.node_count (after update)")
	require.Equalf(t, nameV2, output(t, cluster, "cluster_name"), "%s mismatch", "cluster_name (after update)")

	// --- IMPORT cluster
	if _, err := ImportClusterPlanApply(
		t,
		token, endpoint, projectID, regionID, cluster.ID,
		cluster.Dir,
		cluster.Opts.RetryableTerraformErrors,
	); err != nil {
		t.Fatalf("failed to import cluster: %v", err)
	}

	if err := cluster.Destroy(t); err != nil {
		t.Fatalf("terraform destroy for cluster: %v", err)
	}
	testSucceed = true
}

func output(t *testing.T, cluster *Cluster, name string) string {
	t.Helper()
	return tt.Output(t, cluster.Opts, name)
}
