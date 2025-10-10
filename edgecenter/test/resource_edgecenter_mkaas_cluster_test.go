package edgecenter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
)

func TestMKaaSCluster_ApplyUpdateImportDestroy(t *testing.T) {
	token := requireEnv(t, "EC_TOKEN")
	endpoint := os.Getenv("EC_API_ENDPOINT")
	projectID := requireEnv(t, "EC_PROJECT_ID")
	regionID := requireEnv(t, "EC_REGION_ID")
	networkID := requireEnv(t, "EC_NETWORK_ID")
	subnetID := requireEnv(t, "EC_SUBNET_ID")
	sshKeypair := requireEnv(t, "EC_SSH_KEYPAIR_NAME")
	flavor := requireEnv(t, "EC_CP_FLAVOR")
	volumeType := os.Getenv("EC_VOLUME_TYPE")
	if volumeType == "" {
		volumeType = "ssd_hiiops"
	}
	version := os.Getenv("EC_K8S_VERSION")
	if version == "" {
		version = "v1.31.0"
	}

	tmp := t.TempDir()
	mainTf := `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token  = var.token
  edgecenter_cloud_api = var.endpoint
}

variable "token"     { type = string }
variable "endpoint"  { type = string }
variable "project_id" { type = number }
variable "region_id"  { type = number }
variable "network_id" { type = string }
variable "subnet_id"  { type = string }
variable "ssh_keypair_name" { type = string }
variable "cp_flavor" { type = string }
variable "cp_node_count" { type = number }
variable "cp_volume_size" { type = number }
variable "cp_volume_type" { type = string }
variable "k8s_version" { type = string }
variable "name" { type = string }

resource "edgecenter_mkaas_cluster" "test" {
  project_id = var.project_id
  region_id  = var.region_id

  name               = var.name
  ssh_keypair_name   = var.ssh_keypair_name
  network_id         = var.network_id
  subnet_id          = var.subnet_id

  control_plane {
    flavor      = var.cp_flavor
    node_count  = var.cp_node_count
    volume_size = var.cp_volume_size
    volume_type = var.cp_volume_type
    version     = var.k8s_version
  }
}

# --- outputs для проверки совпадений ---
output "cluster_id"            { value = edgecenter_mkaas_cluster.test.id }
output "cluster_name"          { value = edgecenter_mkaas_cluster.test.name }
output "out_project_id"        { value = tostring(edgecenter_mkaas_cluster.test.project_id) }
output "out_region_id"         { value = tostring(edgecenter_mkaas_cluster.test.region_id) }
output "out_network_id"        { value = edgecenter_mkaas_cluster.test.network_id }
output "out_subnet_id"         { value = edgecenter_mkaas_cluster.test.subnet_id }
output "out_ssh_keypair_name"  { value = edgecenter_mkaas_cluster.test.ssh_keypair_name }
output "out_cp_flavor"         { value = edgecenter_mkaas_cluster.test.control_plane[0].flavor }
output "out_cp_node_count"     { value = edgecenter_mkaas_cluster.test.control_plane[0].node_count }
output "out_cp_volume_size"    { value = edgecenter_mkaas_cluster.test.control_plane[0].volume_size }
output "out_cp_volume_type"    { value = edgecenter_mkaas_cluster.test.control_plane[0].volume_type }
output "out_k8s_version"       { value = edgecenter_mkaas_cluster.test.control_plane[0].version }

# полезные вычисляемые
output "internal_ip"           { value = edgecenter_mkaas_cluster.test.internal_ip }
output "external_ip"           { value = edgecenter_mkaas_cluster.test.external_ip }
output "state"                 { value = edgecenter_mkaas_cluster.test.state }
output "created"               { value = edgecenter_mkaas_cluster.test.created }
`
	if err := os.WriteFile(filepath.Join(tmp, "main.tf"), []byte(mainTf), 0644); err != nil {
		t.Fatalf("write main.tf: %v", err)
	}

	baseName := "tf-mkaas-" + strings.ToLower(random.UniqueId())
	nameV1 := baseName

	tfOpts := &tt.Options{
		TerraformDir: tmp,
		NoColor:      true,
		EnvVars: map[string]string{
			"EDGECENTER_TOKEN": token,
		},
		Vars: map[string]interface{}{
			"token":            token,
			"endpoint":         endpoint,
			"project_id":       mustAtoi(projectID),
			"region_id":        mustAtoi(regionID),
			"network_id":       networkID,
			"subnet_id":        subnetID,
			"ssh_keypair_name": sshKeypair,
			"cp_flavor":        flavor,
			"cp_node_count":    1,
			"cp_volume_size":   40,
			"cp_volume_type":   volumeType,
			"k8s_version":      version,
			"name":             nameV1,
		},
		RetryableTerraformErrors: map[string]string{
			".*429.*":              "rate-limit",
			".*timeout.*":          "transient network",
			".*connection reset.*": "transient network",
		},
	}

	defer func() { tt.Destroy(t, tfOpts) }()

	// create
	tt.Apply(t, tfOpts)

	// --- проверки "что записали — то и прочитали" после создания ---
	assertEq(t, tt.Output(t, tfOpts, "cluster_name"), nameV1, "cluster_name")
	assertEq(t, tt.Output(t, tfOpts, "out_project_id"), projectID, "project_id")
	assertEq(t, tt.Output(t, tfOpts, "out_region_id"), regionID, "region_id")
	assertEq(t, tt.Output(t, tfOpts, "out_network_id"), networkID, "network_id")
	assertEq(t, tt.Output(t, tfOpts, "out_subnet_id"), subnetID, "subnet_id")
	assertEq(t, tt.Output(t, tfOpts, "out_ssh_keypair_name"), sshKeypair, "ssh_keypair_name")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_flavor"), flavor, "control_plane.flavor")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_volume_type"), volumeType, "control_plane.volume_type")
	assertEq(t, tt.Output(t, tfOpts, "out_k8s_version"), version, "control_plane.version")

	assertEq(t, tt.Output(t, tfOpts, "out_cp_node_count"), "1", "control_plane.node_count")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_volume_size"), "40", "control_plane.volume_size")

	id := tt.Output(t, tfOpts, "cluster_id")
	if id == "" {
		t.Fatalf("cluster_id is empty after create")
	}
	if tt.Output(t, tfOpts, "state") == "" {
		t.Fatalf("state is empty after create")
	}
	if tt.Output(t, tfOpts, "created") == "" {
		t.Fatalf("created is empty after create")
	}
	intIP := tt.Output(t, tfOpts, "internal_ip")
	extIP := tt.Output(t, tfOpts, "external_ip")
	if intIP == "" && extIP == "" {
		t.Log("warning: both internal_ip and external_ip are empty; may be OK for private clusters")
	}

	// update node_count
	tfOpts.Vars["cp_node_count"] = 3
	tt.Apply(t, tfOpts)

	assertEq(t, tt.Output(t, tfOpts, "out_cp_node_count"), "3", "control_plane.node_count (after update)")

	// import check
	importDir := filepath.Join(tmp, "import")
	if err := os.MkdirAll(importDir, 0755); err != nil {
		t.Fatalf("mkdir import: %v", err)
	}

	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(mainTf), 0644); err != nil {
		t.Fatalf("write import/main.tf: %v", err)
	}

	importID := strings.Join([]string{projectID, regionID, id}, ":")

	baseTf := `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token  = var.token
  edgecenter_cloud_api = var.endpoint
}

import {
  to = edgecenter_mkaas_cluster.test
  id = "` + importID + `"
}

variable "token"            { type = string }
variable "endpoint"         { type = string }
`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(baseTf), 0644); err != nil {
		t.Fatalf("write import/main.tf: %v", err)
	}

	importOpts := &tt.Options{
		TerraformDir: importDir,
		NoColor:      true,
		EnvVars: map[string]string{
			"EDGECENTER_TOKEN": token,
		},
		Vars: map[string]interface{}{
			"token":    token,
			"endpoint": endpoint,
		},
		RetryableTerraformErrors: tfOpts.RetryableTerraformErrors,
	}
	tt.RunTerraformCommand(
		t, importOpts,
		"plan",
		"-generate-config-out=generated.tf",
		"-var", "token="+token,
		"-var", "endpoint="+endpoint,
		"-input=false",
		"-lock-timeout=5m",
	)
	if _, err := os.Stat(filepath.Join(importDir, "generated.tf")); err != nil {
		t.Fatalf("generated.tf not found after plan -generate-config-out: %v", err)
	}
	tt.RunTerraformCommand(
		t, importOpts,
		"apply",
		"-auto-approve",
		"-var", "token="+token,
		"-var", "endpoint="+endpoint,
		"-input=false",
		"-lock-timeout=5m",
	)

	if out, err := tt.RunTerraformCommandE(
		t, importOpts,
		"plan",
		"-detailed-exitcode",
		"-var", "token="+token,
		"-var", "endpoint="+endpoint,
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		t.Fatalf("terraform plan after import/apply is not empty (err=%v)\n%s", err, out)
	}
}

func assertEq(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch: got %q, want %q", field, got, want)
	}
}

func mustAtoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return val
}
