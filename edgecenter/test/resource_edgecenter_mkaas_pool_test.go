package edgecenter_test

import (
	"net"
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

	testFailed := true
	var cluster *Cluster
	var networkID string
	var subnetID string
	var keypairID string
	var client *edgecloudV2.Client

	t.Cleanup(func() {
		if client == nil {
			return
		}

		if cluster != nil && testFailed {
			if err := DeleteTestMKaaSCluster(t, client, cluster.ID); err != nil {
				t.Fatalf("cleanup failed: delete cluster %s via API: %v", cluster.ID, err)
			}
			cluster = nil
		}

		if subnetID != "" {
			if err := DeleteTestSubnet(client, subnetID); err != nil {
				t.Fatalf("cleanup failed: delete subnet %s: %v", subnetID, err)
			}
		}

		if networkID != "" {
			if err := DeleteTestNetwork(client, networkID); err != nil {
				t.Fatalf("cleanup failed: delete network %s: %v", networkID, err)
			}
		}

		if keypairID != "" {
			if err := DeleteTestKeypair(t, client, keypairID); err != nil {
				t.Fatalf("cleanup failed: delete SSH keypair %s: %v", keypairID, err)
			}
		}
	})

	// --- env
	t.Log("Reading environment variables...")
	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	endpoint := os.Getenv("EC_API")
	if endpoint == "" {
		endpoint = "https://api.edgecenter.online"
	}
	t.Logf("Using endpoint: %s", endpoint)
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

	base := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	keypairName := base + "-key"
	var err error
	client, err = CreateClient(t, token, endpoint, projectID, regionID)
	require.NoError(t, err, "failed to create keypair client")

	t.Logf("Creating SSH keypair with name: %s", keypairName)
	keypairID, err = CreateTestKeypair(t, client, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Logf("SSH keypair created successfully with ID: %s, name: %s", keypairID, keypairName)
	sshKeypair := keypairName

	// Create network and subnet dynamically
	t.Log("Creating network...")
	networkName := base + "-net"
	t.Logf("Creating network with name: %s", networkName)
	networkID, err = CreateTestNetwork(client, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	require.NoError(t, err, "failed to create network")
	t.Logf("Network created successfully with ID: %s", networkID)

	t.Log("Creating subnet...")
	subnetName := base + "-subnet"
	t.Logf("Creating subnet with name: %s in network: %s", subnetName, networkID)
	ip := net.ParseIP("192.168.42.1")
	subnetID, err = CreateTestSubnet(client, &edgecloudV2.SubnetworkCreateRequest{
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
	cluster, err = CreateCluster(t, tfData{
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

	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	cl := cluster
	require.NoError(t, err, "failed to create cluster")
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
  id = "` + strings.Join([]string{projectID, regionID, poolID, cl.ID}, ":") + `"
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
	cluster = nil
	testFailed = false
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
