//go:build cloud_data_source_mkaas

package edgecenter_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// HCL для data source пула
const dataSourcePoolMainTmpl = `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token  = "{{ .Token }}"
}

data "edgecenter_mkaas_pool" "acctest" {
  project_id = {{ .ProjectID }}
  region_id  = {{ .RegionID }}
  cluster_id = {{ .ClusterID }}
  pool_id    = {{ .PoolID }}
}

output "pool_id"           { value = data.edgecenter_mkaas_pool.acctest.id }
output "pool_name"         { value = data.edgecenter_mkaas_pool.acctest.name }
output "out_cluster_id"    { value = tostring(data.edgecenter_mkaas_pool.acctest.cluster_id) }
output "out_flavor"        { value = data.edgecenter_mkaas_pool.acctest.flavor }
output "out_node_count"    { value = tostring(data.edgecenter_mkaas_pool.acctest.node_count) }
output "out_volume_size"   { value = tostring(data.edgecenter_mkaas_pool.acctest.volume_size) }
output "out_volume_type"   { value = data.edgecenter_mkaas_pool.acctest.volume_type }
output "out_state"         { value = data.edgecenter_mkaas_pool.acctest.state }
output "out_status"        { value = data.edgecenter_mkaas_pool.acctest.status }
output "out_security_group_ids" { value = data.edgecenter_mkaas_pool.acctest.security_group_ids }
output "out_label_env"     { value = data.edgecenter_mkaas_pool.acctest.labels["env"] }
`

type dataSourcePoolTfData struct {
	Token     string
	ProjectID string
	RegionID  string
	ClusterID string
	PoolID    string
}

func TestAccDataSourceMKaaSPool(t *testing.T) {

	t.Log("Starting TestAccDataSourceMKaaSPool")

	// --- env
	t.Log("Reading environment variables...")
	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	cloudAPIURL := requireEnv(t, "EC_API")
	projectID := requireEnv(t, "TEST_PROJECT_ID")
	regionID := requireEnv(t, "TEST_MKAAS_REGION_ID")

	base := "tf-mkaas-ds-" + strings.ToLower(random.UniqueId())
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

	// Create a security group
	t.Log("Creating security group...")
	sgName := base + "-sg"
	sg, _, err := client.SecurityGroups.Create(context.Background(), &edgecloudV2.SecurityGroupCreateRequest{
		SecurityGroup: edgecloudV2.SecurityGroupCreateRequestInner{
			Name: sgName,
		},
	})
	require.NoError(t, err, "failed to create security group")
	t.Logf("Security group created successfully with ID: %s", sg.ID)
	t.Cleanup(func() {
		time.Sleep(30 * time.Second)
		if _, err := client.SecurityGroups.Delete(context.Background(), sg.ID); err != nil {
			t.Errorf("cleanup failed: delete security group %s: %v", sg.ID, err)
		}
	})

	// Create cluster
	t.Log("Creating cluster...")
	clusterName := base + "-cls"
	cluster, err := CreateCluster(t, tfData{
		Token:                    token,
		ProjectID:                projectID,
		RegionID:                 regionID,
		NetworkID:                networkID,
		SubnetID:                 subnetID,
		PodSubnet:                podSubnet,
		ServiceSubnet:            serviceSubnet,
		PublishKubeApiToInternet: false,
		SSHKeypair:               keypairName,
		Name:                     clusterName,
		CPFlavor:                 masterFlavor,
		CPNodeCount:              1,
		CPVolumeSize:             30,
		CPVolumeType:             workerVolumeType,
		CPVersion:                kubernetesVersion,
	})
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	require.NoError(t, err, "failed to create cluster")
	t.Logf("Cluster created successfully with ID: %s", cluster.ID)
	var testSucceed bool
	t.Cleanup(func() {
		if cluster != nil && !testSucceed {
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

	poolName := base + "-pool"
	poolData := poolTfData{
		Token:            token,
		ProjectID:        cluster.Data.ProjectID,
		RegionID:         cluster.Data.RegionID,
		ClusterID:        cluster.ID,
		Name:             poolName,
		Flavor:           masterFlavor,
		NodeCount:        1,
		VolumeSize:       30,
		VolumeType:       workerVolumeType,
		SecurityGroupIDs: []string{sg.ID},
		Labels: map[string]string{
			"env": "test",
		},
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

	// Check pool was created successfully
	poolID := tt.Output(t, poolOpts, "pool_id")
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("pool_id is empty after create")
	}
	t.Logf("Pool created successfully with ID: %s", poolID)

	// Test data source
	t.Log("Testing data source...")
	dataSourceDir := filepath.Join(cluster.Dir, "data-source")
	if err := os.MkdirAll(dataSourceDir, 0755); err != nil {
		t.Fatalf("mkdir data-source dir: %v", err)
	}
	dataSourceMain := filepath.Join(dataSourceDir, "main.tf")

	dataSourceData := dataSourcePoolTfData{
		Token:     token,
		ProjectID: cluster.Data.ProjectID,
		RegionID:  cluster.Data.RegionID,
		ClusterID: cluster.ID,
		PoolID:    poolID,
	}
	err = renderTemplateToWith(dataSourceMain, dataSourcePoolMainTmpl, dataSourceData)
	if err != nil {
		t.Fatalf("write data-source main.tf: %v", err)
	}

	dataSourceOpts := &tt.Options{
		TerraformDir: dataSourceDir,
		NoColor:      true,
	}

	if _, err := tt.ApplyAndIdempotentE(t, dataSourceOpts); err != nil {
		t.Fatalf("terraform apply (data-source): %v", err)
	}

	// Check data source outputs
	require.Equalf(t, poolID, tt.Output(t, dataSourceOpts, "pool_id"), "%s mismatch", "pool_id")
	require.Equalf(t, poolName, tt.Output(t, dataSourceOpts, "pool_name"), "%s mismatch", "pool_name")
	require.Equalf(t, cluster.ID, tt.Output(t, dataSourceOpts, "out_cluster_id"), "%s mismatch", "cluster_id")
	require.Equalf(t, masterFlavor, tt.Output(t, dataSourceOpts, "out_flavor"), "%s mismatch", "flavor")
	require.Equalf(t, "1", tt.Output(t, dataSourceOpts, "out_node_count"), "%s mismatch", "node_count")
	require.Equalf(t, "30", tt.Output(t, dataSourceOpts, "out_volume_size"), "%s mismatch", "volume_size")
	require.Equalf(t, workerVolumeType, tt.Output(t, dataSourceOpts, "out_volume_type"), "%s mismatch", "volume_type")
	require.Equalf(t, "["+sg.ID+"]", tt.Output(t, dataSourceOpts, "out_security_group_ids"), "%s mismatch", "security_group_ids")
	_ = tt.Output(t, dataSourceOpts, "out_state")
	_ = tt.Output(t, dataSourceOpts, "out_status")

	if err := cluster.Destroy(t); err != nil {
		t.Fatalf("terraform destroy for cluster: %v", err)
	}
	testSucceed = true
}
