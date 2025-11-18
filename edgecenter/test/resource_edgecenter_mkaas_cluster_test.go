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
	MKaaSRegionId   = "2403"
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
	regionID := MKaaSRegionId //TODO: when 8 region will be strong - requireEnv(t, "TEST_REGION_ID")

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

	// Create keypair client
	t.Log("Creating keypair client...")
	keypairClient, err := CreateKeypairClient(t, token, endpoint, projectID)
	require.NoError(t, err, "failed to create keypair client")
	t.Log("Keypair client created successfully")

	// Create network and subnet clients
	t.Log("Creating network and subnet clients...")
	networkClient, err := CreateNetworkAndSubnetClients(t, token, endpoint, projectID, regionID)
	require.NoError(t, err, "failed to create network and subnet clients")
	t.Log("Network and subnet clients created successfully")

	// Create SSH keypair dynamically
	baseName := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	keypairName := baseName + "-key"
	t.Logf("Creating SSH keypair with name: %s", keypairName)
	keypairID, err := CreateTestKeypair(t, keypairClient, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Logf("SSH keypair created successfully with ID: %s, name: %s", keypairID, keypairName)
	sshKeypair := keypairName

	// Create network and subnet dynamically
	t.Log("Creating network...")
	networkName := baseName + "-net"
	t.Logf("Creating network with name: %s", networkName)
	networkID, err := CreateTestNetwork(networkClient, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	require.NoError(t, err, "failed to create network")
	t.Logf("Network created successfully with ID: %s", networkID)

	t.Log("Creating subnet...")
	subnetName := baseName + "-subnet"
	t.Logf("Creating subnet with name: %s in network: %s", subnetName, networkID)
	ip := net.ParseIP("192.168.42.1")
	subnetID, err := CreateTestSubnet(networkClient, &edgecloudV2.SubnetworkCreateRequest{
		Name:                   subnetName,
		NetworkID:              networkID,
		CIDR:                   "192.168.42.0/24",
		EnableDHCP:             true,
		ConnectToNetworkRouter: true,
		GatewayIP:              &ip,
	})
	require.NoError(t, err, "failed to create subnet")
	t.Logf("Subnet created successfully with ID: %s", subnetID)

	var cleanupNetworkID = networkID
	t.Cleanup(func() {
		if err := DeleteTestNetwork(networkClient, cleanupNetworkID); err != nil {
			t.Logf("failed to delete network %s: %v", cleanupNetworkID, err)
		}
	})

	var cleanupSubnetID = subnetID
	t.Cleanup(func() {
		if err := DeleteTestSubnet(networkClient, cleanupSubnetID); err != nil {
			t.Logf("failed to delete subnet %s: %v", cleanupSubnetID, err)
		}
	})

	var cleanupKeypairID = keypairID
	t.Cleanup(func() {
		if err := DeleteTestKeypair(t, keypairClient, cleanupKeypairID); err != nil {
			t.Logf("failed to delete SSH keypair %s: %v", cleanupKeypairID, err)
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
		SSHKeypair:   sshKeypair,
		Name:         nameV1,
		CPFlavor:     cpFlavor,
		CPNodeCount:  1,
		CPVolumeSize: 30,
		CPVolumeType: cpVolumeType,
		CPVersion:    cpVersion,
	}

	// --- CREATE cluster
	cl := CreateCluster(t, data)

	// Check cluster
	require.Equalf(t, cl.ID, output(t, cl, "cluster_id"), "%s mismatch", "cluster_id non-empty")
	require.Equalf(t, nameV1, output(t, cl, "cluster_name"), "%s mismatch", "cluster_name")
	require.Equalf(t, projectID, output(t, cl, "out_project_id"), "%s mismatch", "project_id")
	require.Equalf(t, regionID, output(t, cl, "out_region_id"), "%s mismatch", "region_id")
	require.Equalf(t, sshKeypair, output(t, cl, "out_ssh_keypair_name"), "%s mismatch", "ssh_keypair_name")
	require.Equalf(t, networkID, output(t, cl, "out_network_id"), "%s mismatch", "network_id")
	require.Equalf(t, subnetID, output(t, cl, "out_subnet_id"), "%s mismatch", "subnet_id")
	require.Equalf(t, cpFlavor, output(t, cl, "out_cp_flavor"), "%s mismatch", "control_plane.flavor")
	require.Equalf(t, "1", output(t, cl, "out_cp_node_count"), "%s mismatch", "control_plane.node_count")
	require.Equalf(t, "30", output(t, cl, "out_cp_volume_size"), "%s mismatch", "control_plane.volume_size")
	require.Equalf(t, cpVolumeType, output(t, cl, "out_cp_volume_type"), "%s mismatch", "control_plane.volume_type")
	require.Equalf(t, cpVersion, output(t, cl, "out_k8s_version"), "%s mismatch", "control_plane.version")

	// --- UPDATE cluster
	cl.UpdateCluster(t, func(d *tfData) {
		d.Name = nameV2
		d.CPNodeCount = 3
	})
	require.Equalf(t, "3", output(t, cl, "out_cp_node_count"), "%s mismatch", "control_plane.node_count (after update)")
	require.Equalf(t, nameV2, output(t, cl, "cluster_name"), "%s mismatch", "cluster_name (after update)")

	// --- IMPORT cluster
	_ = ImportClusterPlanApply(
		t,
		token, endpoint, projectID, regionID, cl.ID,
		cl.Dir,
		cl.Opts.RetryableTerraformErrors,
	)

	// Create MKaaS client for cluster deletion
	mkaasClient, err := CreateMKaaSClient(t, token, endpoint, projectID, regionID)
	require.NoError(t, err, "failed to create MKaaS client")
	t.Log("MKaaS client created successfully")
	t.Log("Deleting cluster via API...")
	err = DeleteTestMKaaSCluster(t, mkaasClient, cl.ID)
	require.NoError(t, err, "failed to delete cluster")
	t.Log("Cluster deleted successfully")

}

func output(t *testing.T, cl *Cluster, name string) string {
	t.Helper()
	return tt.Output(t, cl.Opts, name)
}
