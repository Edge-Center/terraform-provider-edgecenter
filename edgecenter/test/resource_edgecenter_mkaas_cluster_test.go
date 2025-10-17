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

type tfData struct {
	Token        string
	Endpoint     string
	ProjectID    string
	RegionID     string
	NetworkID    string
	SubnetID     string
	SSHKeypair   string
	Name         string
	CPFlavor     string
	CPNodeCount  int
	CPVolumeSize int
	CPVolumeType string
	CPVersion    string
}

const mainTmpl = `
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

resource "edgecenter_mkaas_cluster" "test" {
  project_id = {{ .ProjectID }}
  region_id  = {{ .RegionID }}

  name               = "{{ .Name }}"
  ssh_keypair_name   = "{{ .SSHKeypair }}"
  network_id         = "{{ .NetworkID }}"
  subnet_id          = "{{ .SubnetID }}"

  control_plane {
    flavor      = "{{ .CPFlavor }}"
    node_count  = {{ .CPNodeCount }}
    volume_size = {{ .CPVolumeSize }}
    volume_type = "{{ .CPVolumeType }}"
    version     = "{{ .CPVersion }}"
  }
}

# --- outputs для проверок ---
output "cluster_id"            { value = edgecenter_mkaas_cluster.test.id }
output "cluster_name"          { value = edgecenter_mkaas_cluster.test.name }
output "out_project_id"        { value = tostring(edgecenter_mkaas_cluster.test.project_id) }
output "out_region_id"         { value = tostring(edgecenter_mkaas_cluster.test.region_id) }
output "out_network_id"        { value = edgecenter_mkaas_cluster.test.network_id }
output "out_subnet_id"         { value = edgecenter_mkaas_cluster.test.subnet_id }
output "out_ssh_keypair_name"  { value = edgecenter_mkaas_cluster.test.ssh_keypair_name }
output "out_cp_flavor"         { value = edgecenter_mkaas_cluster.test.control_plane[0].flavor }
output "out_cp_node_count"     { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].node_count) }
output "out_cp_volume_size"    { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].volume_size) }
output "out_cp_volume_type"    { value = edgecenter_mkaas_cluster.test.control_plane[0].volume_type }
output "out_k8s_version"       { value = edgecenter_mkaas_cluster.test.control_plane[0].version }

# вычисляемые
output "internal_ip"           { value = edgecenter_mkaas_cluster.test.internal_ip }
output "external_ip"           { value = edgecenter_mkaas_cluster.test.external_ip }
output "state"                 { value = edgecenter_mkaas_cluster.test.state }
output "created"               { value = edgecenter_mkaas_cluster.test.created }
`

func TestMKaaSCluster_ApplyUpdateImportDestroy(t *testing.T) {
	if os.Getenv("RUN_MKAAS_IT") != "1" {
		t.Skip("This test requires MKAAS_IT=1")
	}
	token := requireEnv(t, "EC_TOKEN")
	endpoint := os.Getenv("EC_API_ENDPOINT")
	projectID := requireEnv(t, "EC_PROJECT_ID")
	regionID := requireEnv(t, "EC_REGION_ID")
	networkID := requireEnv(t, "EC_NETWORK_ID")
	subnetID := requireEnv(t, "EC_SUBNET_ID")
	sshKeypair := requireEnv(t, "EC_SSH_KEYPAIR_NAME")
	cpFlavor := requireEnv(t, "EC_CP_FLAVOR")
	cpNodeCount := "1"
	cpVolumeSize := "30"
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

	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.tf")
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
	if err := renderTemplateTo(mainPath, data); err != nil {
		t.Fatalf("write main.tf (create): %v", err)
	}

	tfOpts := &tt.Options{
		TerraformDir: tmp,
		NoColor:      true,
		RetryableTerraformErrors: map[string]string{
			".*429.*":              "rate-limit",
			".*timeout.*":          "transient network",
			".*connection reset.*": "transient network",
		},
	}

	defer func() { tt.Destroy(t, tfOpts) }()

	// create
	tt.ApplyAndIdempotent(t, tfOpts)

	// --- проверки "что записали — то и прочитали" после создания ---
	assertEq(t, tt.Output(t, tfOpts, "cluster_name"), nameV1, "cluster_name")
	assertEq(t, tt.Output(t, tfOpts, "out_project_id"), projectID, "project_id")
	assertEq(t, tt.Output(t, tfOpts, "out_region_id"), regionID, "region_id")
	assertEq(t, tt.Output(t, tfOpts, "out_ssh_keypair_name"), sshKeypair, "ssh_keypair_name")
	assertEq(t, tt.Output(t, tfOpts, "out_network_id"), networkID, "network_id")
	assertEq(t, tt.Output(t, tfOpts, "out_subnet_id"), subnetID, "subnet_id")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_flavor"), cpFlavor, "control_plane.flavor")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_node_count"), cpNodeCount, "control_plane.node_count")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_volume_size"), cpVolumeSize, "control_plane.volume_size")
	assertEq(t, tt.Output(t, tfOpts, "out_cp_volume_type"), cpVolumeType, "control_plane.volume_type")
	assertEq(t, tt.Output(t, tfOpts, "out_k8s_version"), cpVersion, "control_plane.version")
	// проверки на computed поля
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
	data.Name = nameV2
	data.CPNodeCount = 3
	if err := renderTemplateTo(mainPath, data); err != nil {
		t.Fatalf("write main.tf (update): %v", err)
	}
	tt.ApplyAndIdempotent(t, tfOpts)

	assertEq(t, tt.Output(t, tfOpts, "out_cp_node_count"), "3", "control_plane.node_count (after update)")
	assertEq(t, tt.Output(t, tfOpts, "cluster_name"), nameV2, "cluster_name (after update)")

	// import check
	importDir := filepath.Join(tmp, "import")
	if err := os.MkdirAll(importDir, 0755); err != nil {
		t.Fatalf("mkdir import: %v", err)
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
  to = edgecenter_mkaas_cluster.test
  id = "` + strings.Join([]string{projectID, regionID, id}, ":") + `"
}

`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0o644); err != nil {
		t.Fatalf("write import/main.tf: %v", err)
	}

	importOpts := &tt.Options{
		TerraformDir:             importDir,
		NoColor:                  true,
		RetryableTerraformErrors: tfOpts.RetryableTerraformErrors,
	}

	tt.RunTerraformCommand(
		t, importOpts,
		"plan",
		"-generate-config-out=generated.tf",
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
		"-input=false",
		"-lock-timeout=5m",
	)

	if out, err := tt.RunTerraformCommandE(
		t, importOpts,
		"plan",
		"-detailed-exitcode",
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

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return val
}

func renderTemplateTo(path string, data tfData) error {
	tpl := template.Must(template.New("main").Parse(mainTmpl))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}
