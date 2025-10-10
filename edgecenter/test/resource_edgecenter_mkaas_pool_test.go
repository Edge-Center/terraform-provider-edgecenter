package edgecenter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
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

func TestMkaasPool_ApplyUpdateImportDestroy(t *testing.T) {
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
	cpFlavor := os.Getenv("EC_POOL_FLAVOR")
	if strings.TrimSpace(cpFlavor) == "" {
		cpFlavor = requireEnv(t, "EC_CP_FLAVOR")
	}
	volType := os.Getenv("EC_VOLUME_TYPE")
	if strings.TrimSpace(volType) == "" {
		volType = "ssd_hiiops"
	}
	k8sVersion := os.Getenv("EC_K8S_VERSION")
	if strings.TrimSpace(k8sVersion) == "" {
		k8sVersion = "v1.31.0"
	}

	// Create cluster
	base := "tf-mkaas-" + strings.ToLower(random.UniqueId())
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
	t.Cleanup(func() { tt.Destroy(t, poolOpts) })

	tt.ApplyAndIdempotent(t, poolOpts)

	// Check pool
	poolID := tt.Output(t, poolOpts, "pool_id")
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("pool_id is empty after create")
	}
	assertEq(t, tt.Output(t, poolOpts, "pool_name"), poolNameV1, "pool_name")
	assertEq(t, tt.Output(t, poolOpts, "out_project_id"), projectID, "project_id")
	assertEq(t, tt.Output(t, poolOpts, "out_region_id"), regionID, "region_id")
	assertEq(t, tt.Output(t, poolOpts, "out_cluster_id"), cl.ID, "cluster_id")
	assertEq(t, tt.Output(t, poolOpts, "out_flavor"), cpFlavor, "flavor")
	assertEq(t, tt.Output(t, poolOpts, "out_node_count"), "1", "node_count")
	assertEq(t, tt.Output(t, poolOpts, "out_volume_size"), "30", "volume_size")
	assertEq(t, tt.Output(t, poolOpts, "out_volume_type"), volType, "volume_type")
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
	assertEq(t, tt.Output(t, poolOpts, "pool_name"), poolNameV2, "pool_name (after update)")
	assertEq(t, tt.Output(t, poolOpts, "out_node_count"), "2", "node_count (after update)")

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
