package edgecenter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// HCL для пула + полезные outputs
const poolMainTmpl = `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token  = "{{ .Token }}"
  edgecenter_cloud_api = "{{ .Endpoint }}"
}

resource "edgecenter_mkaas_pool" "np" {
  project_id   = {{ .ProjectID }}
  region_id    = {{ .RegionID }}
  cluster_id   = {{ .ClusterID }}

  name         = "{{ .Name }}"
  flavor       = "{{ .Flavor }}"
  node_count   = {{ .NodeCount }}
  volume_size  = {{ .VolumeSize }}
  volume_type  = "{{ .VolumeType }}"
}

output "pool_id"           { value = edgecenter_mkaas_pool.np.id }
output "pool_name"         { value = edgecenter_mkaas_pool.np.name }
output "out_project_id"    { value = tostring(edgecenter_mkaas_pool.np.project_id) }
output "out_region_id"     { value = tostring(edgecenter_mkaas_pool.np.region_id) }
output "out_cluster_id"    { value = tostring(edgecenter_mkaas_pool.np.cluster_id) }
output "out_flavor"        { value = edgecenter_mkaas_pool.np.flavor }
output "out_node_count"    { value = tostring(edgecenter_mkaas_pool.np.node_count) }
output "out_volume_size"   { value = tostring(edgecenter_mkaas_pool.np.volume_size) }
output "out_volume_type"   { value = edgecenter_mkaas_pool.np.volume_type }
output "out_state"         { value = edgecenter_mkaas_pool.np.state }
output "out_status"        { value = edgecenter_mkaas_pool.np.status }
`

type poolTfData struct {
	Token      string
	Endpoint   string
	ProjectID  string
	RegionID   string
	ClusterID  string
	Name       string
	Flavor     string
	NodeCount  int
	VolumeSize int
	VolumeType string
}

func TestMKaaSPool_ApplyUpdateImportDestroy(t *testing.T) {
	if os.Getenv("RUN_MKAAS_IT") != "1" {
		t.Skip("This test requires RUN_MKAAS_IT=1")
	}

	t.Log("Starting TestMKaaSPool_ApplyUpdateImportDestroy")

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

	volType := os.Getenv("EC_MKAAS_VOLUME_TYPE")
	if volType == "" {
		volType = MKaaSVolumeType
	}

	k8sVersion := os.Getenv("EC_MKAAS_K8S_VERSION")
	if k8sVersion == "" {
		k8sVersion = MKaaSK8sVersion
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

	// Create MKaaS client for cluster deletion
	mkaasClient, err := CreateMKaaSClient(t, token, endpoint, projectID, regionID)
	require.NoError(t, err, "failed to create MKaaS client")
	t.Log("MKaaS client created successfully")

	// Create SSH keypair dynamically
	base := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	keypairName := base + "-key"
	t.Logf("Creating SSH keypair with name: %s", keypairName)
	keypairID, err := CreateTestKeypair(t, keypairClient, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Logf("SSH keypair created successfully with ID: %s, name: %s", keypairID, keypairName)
	sshKeypair := keypairName

	// Create network and subnet dynamically
	t.Log("Creating network...")
	networkName := base + "-net"
	t.Logf("Creating network with name: %s", networkName)
	networkID, err := CreateTestNetwork(networkClient, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	require.NoError(t, err, "failed to create network")
	t.Logf("Network created successfully with ID: %s", networkID)

	t.Log("Creating subnet...")
	subnetName := base + "-subnet"
	t.Logf("Creating subnet with name: %s in network: %s", subnetName, networkID)
	subnetID, err := CreateTestSubnet(networkClient, &edgecloudV2.SubnetworkCreateRequest{
		Name:                   subnetName,
		NetworkID:              networkID,
		CIDR:                   "192.168.42.0/24",
		EnableDHCP:             true,
		ConnectToNetworkRouter: true,
	})
	require.NoError(t, err, "failed to create subnet")
	t.Logf("Subnet created successfully with ID: %s", subnetID)

	// Store resources for cleanup (will be deleted after cluster)
	var cleanupNetworkID = networkID
	var cleanupSubnetID = subnetID
	var cleanupKeypairID = keypairID

	// Create cluster
	t.Log("Creating cluster...")
	clusterName := base + "-cls"
	cl := CreateCluster(t, tfData{
		Token:        token,
		Endpoint:     endpoint,
		ProjectID:    projectID,
		RegionID:     regionID,
		NetworkID:    networkID,
		SubnetID:     subnetID,
		SSHKeypair:   sshKeypair,
		Name:         clusterName,
		CPFlavor:     cpFlavor,
		CPNodeCount:  1,
		CPVolumeSize: 30,
		CPVolumeType: volType,
		CPVersion:    k8sVersion,
	})
	t.Logf("Cluster created successfully with ID: %s", cl.ID)

	// Create pool
	poolDir := filepath.Join(cl.Dir, "pool")
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		t.Fatalf("mkdir pool dir: %v", err)
	}
	poolMain := filepath.Join(poolDir, "main.tf")

	poolNameV1 := base + "-pool-v1"
	poolData := poolTfData{
		Token:      token,
		Endpoint:   endpoint,
		ProjectID:  cl.Data.ProjectID,
		RegionID:   cl.Data.RegionID,
		ClusterID:  cl.ID,
		Name:       poolNameV1,
		Flavor:     cpFlavor,
		NodeCount:  1,
		VolumeSize: 30,
		VolumeType: volType,
	}
	if err := renderTemplateToWith(poolMain, poolMainTmpl, poolData); err != nil {
		t.Fatalf("write pool main.tf (create): %v", err)
	}

	poolOpts := &tt.Options{
		TerraformDir: poolDir,
		NoColor:      true,
		RetryableTerraformErrors: map[string]string{
			".*429.*":              "rate-limit",
			".*timeout.*":          "transient network",
			".*connection reset.*": "transient network",
		},
	}
	// Note: pool will be destroyed when cluster is deleted, so no cleanup needed here

	tt.ApplyAndIdempotent(t, poolOpts)

	// Check pool
	poolID := tt.Output(t, poolOpts, "pool_id")
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("pool_id is empty after create")
	}
	require.Equalf(t, poolNameV1, tt.Output(t, poolOpts, "pool_name"), "%s mismatch", "pool_name")
	require.Equalf(t, projectID, tt.Output(t, poolOpts, "out_project_id"), "%s mismatch", "project_id")
	require.Equalf(t, regionID, tt.Output(t, poolOpts, "out_region_id"), "%s mismatch", "region_id")
	require.Equalf(t, cl.ID, tt.Output(t, poolOpts, "out_cluster_id"), "%s mismatch", "cluster_id")
	require.Equalf(t, cpFlavor, tt.Output(t, poolOpts, "out_flavor"), "%s mismatch", "flavor")
	require.Equalf(t, "1", tt.Output(t, poolOpts, "out_node_count"), "%s mismatch", "node_count")
	require.Equalf(t, "30", tt.Output(t, poolOpts, "out_volume_size"), "%s mismatch", "volume_size")
	require.Equalf(t, volType, tt.Output(t, poolOpts, "out_volume_type"), "%s mismatch", "volume_type")
	_ = tt.Output(t, poolOpts, "out_state")
	_ = tt.Output(t, poolOpts, "out_status")

	// UPDATE pool
	poolNameV2 := base + "-pool-v2"
	poolData.Name = poolNameV2
	poolData.NodeCount = 2
	if err := renderTemplateToWith(poolMain, poolMainTmpl, poolData); err != nil {
		t.Fatalf("write pool main.tf (update): %v", err)
	}
	tt.ApplyAndIdempotent(t, poolOpts)
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
  edgecenter_cloud_api = "` + endpoint + `"
}

import {
  to = edgecenter_mkaas_pool.np
  id = "` + strings.Join([]string{projectID, regionID, poolID, cl.ID}, ":") + `"
}
`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0644); err != nil {
		t.Fatalf("write pool import main.tf: %v", err)
	}
	importOpts := &tt.Options{
		TerraformDir:             importDir,
		NoColor:                  true,
		RetryableTerraformErrors: poolOpts.RetryableTerraformErrors,
	}
	tt.RunTerraformCommand(
		t, importOpts,
		"plan",
		"-generate-config-out=generated.tf",
		"-input=false",
		"-lock-timeout=5m",
	)
	if _, err := os.Stat(filepath.Join(importDir, "generated.tf")); err != nil {
		t.Fatalf("pool generated.tf not found after plan -generate-config-out: %v", err)
	}
	tt.ApplyAndIdempotent(t, importOpts)
	if out, err := tt.RunTerraformCommandE(
		t, importOpts,
		"plan",
		"-detailed-exitcode",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		t.Fatalf("terraform plan for pool after import/apply is not empty (err=%v)\n%s", err, out)
	}

	// --- DELETE cluster via API (before cleanup of network/subnet/keypair)
	t.Log("Deleting cluster via API...")
	err = DeleteTestMKaaSCluster(t, mkaasClient, cl.ID)
	require.NoError(t, err, "failed to delete cluster")
	t.Log("Cluster deleted successfully")

	// Cleanup network, subnet and keypair after cluster is deleted
	t.Cleanup(func() {
		if err := DeleteTestSubnet(networkClient, cleanupSubnetID); err != nil {
			t.Logf("failed to delete subnet %s: %v", cleanupSubnetID, err)
		}
	})
	t.Cleanup(func() {
		if err := DeleteTestNetwork(networkClient, cleanupNetworkID); err != nil {
			t.Logf("failed to delete network %s: %v", cleanupNetworkID, err)
		}
	})
	t.Cleanup(func() {
		if err := DeleteTestKeypair(t, keypairClient, cleanupKeypairID); err != nil {
			t.Logf("failed to delete SSH keypair %s: %v", cleanupKeypairID, err)
		}
	})
}

func renderTemplateToWith(path, tmpl string, data any) error {
	tpl := template.Must(template.New("pool").Parse(tmpl))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}
