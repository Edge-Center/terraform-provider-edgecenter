package edgecenter_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
)

func TestMKaaSCluster_ApplyUpdateImportDestroy(t *testing.T) {
	if os.Getenv("RUN_MKAAS_IT") != "1" {
		t.Skip("This test requires RUN_MKAAS_IT=1")
	}

	// --- env
	token := requireEnv(t, "EC_TOKEN")
	endpoint := os.Getenv("EC_API_ENDPOINT")
	projectID := requireEnv(t, "EC_PROJECT_ID")
	regionID := requireEnv(t, "EC_REGION_ID")
	networkID := requireEnv(t, "EC_NETWORK_ID")
	subnetID := requireEnv(t, "EC_SUBNET_ID")
	sshKeypair := requireEnv(t, "EC_SSH_KEYPAIR_NAME")
	cpFlavor := requireEnv(t, "EC_CP_FLAVOR")

	cpVolumeType := os.Getenv("EC_VOLUME_TYPE")
	if cpVolumeType == "" {
		cpVolumeType = "ssd_hiiops"
	}
	cpVersion := os.Getenv("EC_K8S_VERSION")
	if cpVersion == "" {
		cpVersion = "v1.31.0"
	}

	baseName := "tf-mkaas-" + strings.ToLower(random.UniqueId())
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
	assertEq(t, output(t, cl, "cluster_id"), cl.ID, "cluster_id non-empty")
	assertEq(t, output(t, cl, "cluster_name"), nameV1, "cluster_name")
	assertEq(t, output(t, cl, "out_project_id"), projectID, "project_id")
	assertEq(t, output(t, cl, "out_region_id"), regionID, "region_id")
	assertEq(t, output(t, cl, "out_ssh_keypair_name"), sshKeypair, "ssh_keypair_name")
	assertEq(t, output(t, cl, "out_network_id"), networkID, "network_id")
	assertEq(t, output(t, cl, "out_subnet_id"), subnetID, "subnet_id")
	assertEq(t, output(t, cl, "out_cp_flavor"), cpFlavor, "control_plane.flavor")
	assertEq(t, output(t, cl, "out_cp_node_count"), "1", "control_plane.node_count")
	assertEq(t, output(t, cl, "out_cp_volume_size"), "30", "control_plane.volume_size")
	assertEq(t, output(t, cl, "out_cp_volume_type"), cpVolumeType, "control_plane.volume_type")
	assertEq(t, output(t, cl, "out_k8s_version"), cpVersion, "control_plane.version")

	// --- UPDATE cluster
	cl.UpdateCluster(t, func(d *tfData) {
		d.Name = nameV2
		d.CPNodeCount = 3
	})
	assertEq(t, output(t, cl, "out_cp_node_count"), "3", "control_plane.node_count (after update)")
	assertEq(t, output(t, cl, "cluster_name"), nameV2, "cluster_name (after update)")

	// --- IMPORT cluster
	_ = ImportClusterPlanApply(
		t,
		token, endpoint, projectID, regionID, cl.ID,
		cl.Dir,
		cl.Opts.RetryableTerraformErrors,
	)
}

func output(t *testing.T, cl *Cluster, name string) string {
	t.Helper()
	return tt.Output(t, cl.Opts, name)
}
