//go:build cloud_resource_mkaas

package edgecenter_test

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func TestMKaaSPool_ApplyUpdateImportDestroy(t *testing.T) {

	t.Log("Starting TestMKaaSPool_ApplyUpdateImportDestroy")

	// --- env
	t.Log("Reading environment variables...")
	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	cloudAPIURL := requireEnv(t, "EC_API")
	projectID := requireEnv(t, "TEST_PROJECT_ID")
	regionID := requireEnv(t, "TEST_MKAAS_REGION_ID")

	base := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	keypairName := base + "-key"
	var err error
	client, err := CreateClient(t, token, cloudAPIURL, projectID, regionID)
	require.NoError(t, err, "failed to create keypair client")

	t.Logf("Creating SSH keypair with name: %s", keypairName)
	keypairID, err := CreateTestKeypair(t, client, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Logf("SSH keypair created successfully with ID: %s, name: %s", keypairID, keypairName)
	t.Cleanup(func() {
		if err := DeleteTestKeypair(t, client, keypairID); err != nil {
			t.Errorf("cleanup failed: delete SSH keypair %s: %v", keypairID, err)
		}
	})

	// Create network and subnet dynamically
	t.Log("Creating network...")
	networkName := base + "-net"
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
	subnetName := base + "-subnet"
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

	// Create cluster
	t.Log("Creating cluster...")
	clusterName := base + "-cls"
	cluster, err := CreateCluster(t, tfData{
		Token:         token,
		ProjectID:     projectID,
		RegionID:      regionID,
		NetworkID:     networkID,
		SubnetID:      subnetID,
		SSHKeypair:    keypairName,
		ServiceSubnet: serviceSubnet,
		PodSubnet:     podSubnet,
		Name:          clusterName,
		CPFlavor:      masterFlavor,
		CPNodeCount:   1,
		CPVolumeSize:  30,
		CPVolumeType:  masterVolumeType,
		CPVersion:     kubernetesVersion,
	})
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	require.NoError(t, err, "failed to create cluster")
	t.Logf("Cluster created successfully with ID: %s", cluster.ID)
	var testSuceed bool
	t.Cleanup(func() {
		if cluster != nil && !testSuceed {
			if err := DeleteTestMKaaSCluster(t, client, cluster.ID); err != nil {
				t.Errorf("cleanup failed: delete cluster %s via API: %v", cluster.ID, err)
			}
		}
	})

	// Create pool
	poolDir := filepath.Join(cluster.Dir, "pool")
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		t.Fatalf("mkdir pool dir: %v", err)
	}
	poolMain := filepath.Join(poolDir, "main.tf")

	poolNameV1 := base + "-pool-v1"
	poolData := poolTfData{
		Token:      token,
		ProjectID:  cluster.Data.ProjectID,
		RegionID:   cluster.Data.RegionID,
		ClusterID:  cluster.ID,
		Name:       poolNameV1,
		Flavor:     masterFlavor,
		NodeCount:  1,
		VolumeSize: 30,
		VolumeType: workerVolumeType,
	}
	err = renderTemplateToWith(poolMain, poolMainTmpl, poolData)
	if err != nil {
		t.Fatalf("write pool main.tf (create): %v", err)
	}

	poolOpts := &tt.Options{
		TerraformDir: poolDir,
		NoColor:      true,
	}
	// Note: pool will be destroyed when cluster is deleted, so no cleanup needed here

	if _, err := tt.ApplyAndIdempotentE(t, poolOpts); err != nil {
		t.Fatalf("terraform apply (pool create): %v", err)
	}

	// Check pool
	poolID := tt.Output(t, poolOpts, "pool_id")
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("pool_id is empty after create")
	}
	require.Equalf(t, poolNameV1, tt.Output(t, poolOpts, "pool_name"), "%s mismatch", "pool_name")
	require.Equalf(t, projectID, tt.Output(t, poolOpts, "out_project_id"), "%s mismatch", "project_id")
	require.Equalf(t, regionID, tt.Output(t, poolOpts, "out_region_id"), "%s mismatch", "region_id")
	require.Equalf(t, cluster.ID, tt.Output(t, poolOpts, "out_cluster_id"), "%s mismatch", "cluster_id")
	require.Equalf(t, masterFlavor, tt.Output(t, poolOpts, "out_flavor"), "%s mismatch", "flavor")
	require.Equalf(t, "1", tt.Output(t, poolOpts, "out_node_count"), "%s mismatch", "node_count")
	require.Equalf(t, "30", tt.Output(t, poolOpts, "out_volume_size"), "%s mismatch", "volume_size")
	require.Equalf(t, workerVolumeType, tt.Output(t, poolOpts, "out_volume_type"), "%s mismatch", "volume_type")
	_ = tt.Output(t, poolOpts, "out_state")
	_ = tt.Output(t, poolOpts, "out_status")

	// UPDATE pool
	poolNameV2 := base + "-pool-v2"
	poolData.Name = poolNameV2
	poolData.NodeCount = 2
	err = renderTemplateToWith(poolMain, poolMainTmpl, poolData)

	if err != nil {
		t.Fatalf("write pool main.tf (update): %v", err)
	}
	if _, err := tt.ApplyAndIdempotentE(t, poolOpts); err != nil {
		t.Fatalf("terraform apply (pool update): %v", err)
	}
	require.Equalf(t, poolNameV2, tt.Output(t, poolOpts, "pool_name"), "%s mismatch", "pool_name (after update)")
	require.Equalf(t, "2", tt.Output(t, poolOpts, "out_node_count"), "%s mismatch", "node_count (after update)")

	// IMPORT pool
	importDir := filepath.Join(poolDir, "import")
	if err := os.MkdirAll(importDir, 0755); err != nil {
		t.Fatalf("mkdir pool import dir: %v", err)
	}
	importMain := `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token  = "` + token + `"
}

import {
  to = edgecenter_mkaas_pool.np
  id = "` + strings.Join([]string{projectID, regionID, poolID, cluster.ID}, ":") + `"
}
`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0644); err != nil {
		t.Fatalf("write pool import main.tf: %v", err)
	}
	importOpts := &tt.Options{
		TerraformDir: importDir,
		NoColor:      true,
	}
	if _, err := tt.RunTerraformCommandE(
		t, importOpts,
		"plan",
		"-generate-config-out=generated.tf",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		t.Fatalf("terraform plan (pool import generate config): %v", err)
	}
	if _, err := os.Stat(filepath.Join(importDir, "generated.tf")); err != nil {
		t.Fatalf("pool generated.tf not found after plan -generate-config-out: %v", err)
	}
	if _, err := tt.ApplyAndIdempotentE(t, importOpts); err != nil {
		t.Fatalf("terraform apply (pool import dir): %v", err)
	}
	if out, err := tt.RunTerraformCommandE(
		t, importOpts,
		"plan",
		"-detailed-exitcode",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		t.Fatalf("terraform plan for pool after import/apply is not empty (err=%v)\n%s", err, out)
	}

	if err := cluster.Destroy(t); err != nil {
		t.Fatalf("terraform destroy for cluster: %v", err)
	}
	testSuceed = true
}
