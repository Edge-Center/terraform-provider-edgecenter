package edgecenter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

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

output "cluster_id"         { value = edgecenter_mkaas_cluster.test.id }
output "cluster_name"       { value = edgecenter_mkaas_cluster.test.name }
output "out_project_id"     { value = tostring(edgecenter_mkaas_cluster.test.project_id) }
output "out_region_id"      { value = tostring(edgecenter_mkaas_cluster.test.region_id) }
output "out_network_id"     { value = edgecenter_mkaas_cluster.test.network_id }
output "out_subnet_id"      { value = edgecenter_mkaas_cluster.test.subnet_id }
output "out_ssh_keypair_name" { value = edgecenter_mkaas_cluster.test.ssh_keypair_name }
output "out_cp_flavor"      { value = edgecenter_mkaas_cluster.test.control_plane[0].flavor }
output "out_cp_node_count"  { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].node_count) }
output "out_cp_volume_size" { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].volume_size) }
output "out_cp_volume_type" { value = edgecenter_mkaas_cluster.test.control_plane[0].volume_type }
output "out_k8s_version"    { value = edgecenter_mkaas_cluster.test.control_plane[0].version }

# computed
output "internal_ip"        { value = edgecenter_mkaas_cluster.test.internal_ip }
output "external_ip"        { value = edgecenter_mkaas_cluster.test.external_ip }
output "state"              { value = edgecenter_mkaas_cluster.test.state }
output "created"            { value = edgecenter_mkaas_cluster.test.created }
`

type Cluster struct {
	Dir      string
	MainPath string
	Opts     *tt.Options
	Data     tfData
	ID       string
}

// CreateCluster
func CreateCluster(t *testing.T, data tfData) *Cluster {
	t.Helper()

	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.tf")

	if err := renderTemplateTo(mainPath, data); err != nil {
		t.Fatalf("write main.tf: %v", err)
	}

	opts := &tt.Options{
		TerraformDir: tmp,
		NoColor:      true,
		RetryableTerraformErrors: map[string]string{
			".*429.*":              "rate-limit",
			".*timeout.*":          "transient network",
			".*connection reset.*": "transient network",
		},
	}

	tt.ApplyAndIdempotent(t, opts)
	id := tt.Output(t, opts, "cluster_id")
	if strings.TrimSpace(id) == "" {
		tt.Destroy(t, opts)
		t.Fatalf("cluster_id is empty after create")
	}

	c := &Cluster{
		Dir:      tmp,
		MainPath: mainPath,
		Opts:     opts,
		Data:     data,
		ID:       id,
	}
	t.Cleanup(func() { tt.Destroy(t, opts) })
	return c
}

// UpdateCluster
func (c *Cluster) UpdateCluster(t *testing.T, mutate func(*tfData)) {
	t.Helper()
	if mutate != nil {
		mutate(&c.Data)
	}
	if err := renderTemplateTo(c.MainPath, c.Data); err != nil {
		t.Fatalf("write main.tf (update): %v", err)
	}
	tt.ApplyAndIdempotent(t, c.Opts)
}

// ImportClusterPlanApply
func ImportClusterPlanApply(t *testing.T, token, endpoint, projectID, regionID, clusterID, workDir string, retry map[string]string) *tt.Options {
	t.Helper()

	importDir := filepath.Join(workDir, "import")
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
  id = "` + strings.Join([]string{projectID, regionID, clusterID}, ":") + `"
}
`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0644); err != nil {
		t.Fatalf("write import/main.tf: %v", err)
	}

	opts := &tt.Options{
		TerraformDir:             importDir,
		NoColor:                  true,
		RetryableTerraformErrors: retry,
	}

	tt.RunTerraformCommand(
		t, opts,
		"plan",
		"-generate-config-out=generated.tf",
		"-input=false",
		"-lock-timeout=5m",
	)
	if _, err := os.Stat(filepath.Join(importDir, "generated.tf")); err != nil {
		t.Fatalf("generated.tf not found after plan -generate-config-out: %v", err)
	}

	tt.ApplyAndIdempotent(t, opts)

	if out, err := tt.RunTerraformCommandE(
		t, opts,
		"plan",
		"-detailed-exitcode",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		t.Fatalf("terraform plan after import/apply is not empty (err=%v)\n%s", err, out)
	}

	return opts
}

// --- common utils

func renderTemplateTo(path string, data tfData) error {
	tpl := template.Must(template.New("main").Parse(mainTmpl))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return val
}

func assertEq(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch: got %q, want %q", field, got, want)
	}
}
