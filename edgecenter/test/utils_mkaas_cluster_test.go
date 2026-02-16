package edgecenter_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	tt "github.com/gruntwork-io/terratest/modules/terraform"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type poolTfData struct { //nolint:unused
	Token            string
	Endpoint         string
	ProjectID        string
	RegionID         string
	ClusterID        string
	Name             string
	Flavor           string
	NodeCount        int
	VolumeSize       int
	VolumeType       string
	SecurityGroupIDs []string
	Labels           map[string]string
}

type tfData struct {
	Token                    string
	Endpoint                 string
	ProjectID                string
	RegionID                 string
	NetworkID                string
	SubnetID                 string
	PodSubnet                string
	ServiceSubnet            string
	PublishKubeApiToInternet bool
	SSHKeypair               string
	Name                     string
	CPFlavor                 string
	CPNodeCount              int
	CPVolumeSize             int
	CPVolumeType             string
	CPVersion                string
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
}

resource "edgecenter_mkaas_cluster" "test" {
  project_id = {{ .ProjectID }}
  region_id  = {{ .RegionID }}

  name               = "{{ .Name }}"
  ssh_keypair_name   = "{{ .SSHKeypair }}"
  network_id         = "{{ .NetworkID }}"
  subnet_id          = "{{ .SubnetID }}"

  pod_subnet         = "{{ .PodSubnet }}"
  service_subnet     = "{{ .ServiceSubnet }}"
  publish_kube_api_to_internet = {{ .PublishKubeApiToInternet }}

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
output "out_pod_subnet"     { value = edgecenter_mkaas_cluster.test.pod_subnet }
output "out_service_subnet" { value = edgecenter_mkaas_cluster.test.service_subnet }
output "out_ssh_keypair_name" { value = edgecenter_mkaas_cluster.test.ssh_keypair_name }
output "out_cp_flavor"      { value = edgecenter_mkaas_cluster.test.control_plane[0].flavor }
output "out_cp_node_count"  { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].node_count) }
output "out_cp_volume_size" { value = tostring(edgecenter_mkaas_cluster.test.control_plane[0].volume_size) }
output "out_cp_volume_type" { value = edgecenter_mkaas_cluster.test.control_plane[0].volume_type }
output "out_k8s_version"    { value = edgecenter_mkaas_cluster.test.control_plane[0].version }

# computed
output "internal_ip"        { value = edgecenter_mkaas_cluster.test.internal_ip }
output "external_ip"        { value = edgecenter_mkaas_cluster.test.external_ip }
output "stage"              { value = edgecenter_mkaas_cluster.test.stage }
output "created"            { value = edgecenter_mkaas_cluster.test.created }
`

//nolint:unused
const (
	podSubnet                 = "10.244.0.0/16"
	serviceSubnet             = "10.96.0.0/12"
	kubernetesVersion         = "v1.31.0"
	masterVolumeType          = "ssd_hiiops"
	workerVolumeType          = "ssd_hiiops"
	masterFlavor              = "g3-standard-2-4"
	workerFlavor              = "g3-standard-2-4"
	clusterWorkCompletedStage = "WORK_COMPLETED"
)

type Cluster struct {
	Dir      string
	MainPath string
	Opts     *tt.Options
	Data     tfData
	ID       string
	once     sync.Once
}

// CreateCluster.
func CreateCluster(t *testing.T, data tfData) (*Cluster, error) {
	t.Helper()

	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.tf")

	if err := renderTemplateTo(mainPath, data); err != nil {
		return nil, fmt.Errorf("write main.tf: %w", err)
	}

	opts := &tt.Options{
		TerraformDir: tmp,
		NoColor:      true,
	}

	if _, err := tt.ApplyAndIdempotentE(t, opts); err != nil {
		t.Logf("terraform apply failed: %v", err)
		if _, destroyErr := tt.DestroyE(t, opts); destroyErr != nil {
			t.Logf("terraform destroy attempt after failed apply returned error: %v", destroyErr)
		}
		return nil, fmt.Errorf("terraform apply: %w", err)
	}

	id, err := tt.OutputRequiredE(t, opts, "cluster_id")
	if err != nil || strings.TrimSpace(id) == "" {
		if _, destroyErr := tt.DestroyE(t, opts); destroyErr != nil {
			t.Logf("terraform destroy attempt after empty cluster_id returned error: %v", destroyErr)
		}
		if err != nil {
			return nil, fmt.Errorf("cluster_id output: %w", err)
		}
		return nil, fmt.Errorf("cluster_id is empty after create")
	}

	return &Cluster{
		Dir:      tmp,
		MainPath: mainPath,
		Opts:     opts,
		Data:     data,
		ID:       id,
	}, nil
}

// Destroy tears down cluster resources once.
func (c *Cluster) Destroy(t *testing.T) error {
	t.Helper()
	var destroyErr error
	c.once.Do(func() {
		_, destroyErr = tt.DestroyE(t, c.Opts)
	})
	return destroyErr
}

// UpdateCluster.
func (c *Cluster) UpdateCluster(t *testing.T, mutate func(*tfData)) error {
	t.Helper()
	if mutate != nil {
		mutate(&c.Data)
	}
	if err := renderTemplateTo(c.MainPath, c.Data); err != nil {
		return fmt.Errorf("write main.tf (update): %w", err)
	}

	if _, err := tt.ApplyE(t, c.Opts); err != nil {
		return fmt.Errorf("terraform apply (update): %w", err)
	}

	return nil
}

// ImportClusterPlanApply.
func ImportClusterPlanApply(t *testing.T, token, projectID, regionID, clusterID, workDir string) (*tt.Options, error) {
	t.Helper()

	importDir := filepath.Join(workDir, "import")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir import: %w", err)
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
  to = edgecenter_mkaas_cluster.test
  id = "` + strings.Join([]string{projectID, regionID, clusterID}, ":") + `"
}
`
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0o600); err != nil {
		return nil, fmt.Errorf("write import/main.tf: %w", err)
	}

	opts := &tt.Options{
		TerraformDir: importDir,
		NoColor:      true,
	}

	if _, err := tt.RunTerraformCommandE(
		t, opts,
		"plan",
		"-generate-config-out=generated.tf",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		return nil, fmt.Errorf("terraform plan (import generate config): %w", err)
	}
	if _, err := os.Stat(filepath.Join(importDir, "generated.tf")); err != nil {
		return nil, fmt.Errorf("generated.tf not found after plan -generate-config-out: %w", err)
	}

	if _, err := tt.ApplyAndIdempotentE(t, opts); err != nil {
		return nil, fmt.Errorf("terraform apply (import dir): %w", err)
	}

	if out, err := tt.RunTerraformCommandE(
		t, opts,
		"plan",
		"-detailed-exitcode",
		"-input=false",
		"-lock-timeout=5m",
	); err != nil {
		return nil, fmt.Errorf("terraform plan after import/apply is not empty (err=%w)\n%s", err, out)
	}

	return opts, nil
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

func requireEnv(t *testing.T, key string) string { //nolint:unused
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return val
}

// CreateTestNetwork создаёт сеть через V2 API.
func CreateTestNetwork(client *edgecloudV2.Client, req *edgecloudV2.NetworkCreateRequest) (string, error) {
	ctx := context.Background()

	results, _, err := client.Networks.Create(ctx, req)
	if err != nil {
		return "", err
	}

	if len(results.Tasks) == 0 {
		return "", fmt.Errorf("no task returned for network creation")
	}
	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID, edgecenter.NetworkCreatingTimeout)
	if err != nil {
		return "", err
	}

	createdNetworksRaw, ok := taskInfo.CreatedResources[edgecenter.NetworksPoint]
	if !ok {
		return "", fmt.Errorf("cannot retrieve Network ID from task info: %s", taskID)
	}

	createdNetworks, ok := createdNetworksRaw.([]interface{})
	if !ok || len(createdNetworks) == 0 {
		return "", fmt.Errorf("unexpected created networks payload for task %s", taskID)
	}

	networkID, ok := createdNetworks[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected network id type %T", createdNetworks[0])
	}

	return networkID, nil
}

// DeleteTestNetwork удаляет сеть через V2 API.
func DeleteTestNetwork(client *edgecloudV2.Client, networkID string) error {
	ctx := context.Background()

	results, _, err := client.Networks.Delete(ctx, networkID)
	if err != nil {
		return err
	}

	if len(results.Tasks) == 0 {
		return fmt.Errorf("no task returned for network deletion")
	}
	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID, edgecenter.NetworkDeletingTimeout)
	if err != nil {
		return err
	}

	if taskInfo.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot delete network with ID: %s", networkID)
	}

	return nil
}

// CreateTestSubnet создаёт подсеть через V2 API.
func CreateTestSubnet(client *edgecloudV2.Client, req *edgecloudV2.SubnetworkCreateRequest) (string, error) {
	ctx := context.Background()

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Subnetworks.Create, req, client, edgecenter.SubnetCreatingTimeout)
	if err != nil {
		return "", err
	}

	if len(taskResult.Subnets) == 0 {
		return "", fmt.Errorf("no subnet ID returned after creation")
	}

	return taskResult.Subnets[0], nil
}

// --- SSH keypair utilities

// Test SSH public key for dynamic keypair creation.
const testSSHPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC1bdbQYquD/swsZpFPXagY9KvhlNUTKYMdhRNtlGglAMgRxJS3Q0V74BNElJtP+UU/AbZD4H2ZAwW3PLLD/maclnLlrA48xg/ez9IhppBop0WADZ/nB4EcvQfR/Db7nHDTZERW6EiiGhV6CkHVasK2sY/WNRXqPveeWUlwCqtSnU90l/s9kQCoEfkM2auO6ppJkVrXbs26vcRclS8KL7Cff4HwdVpV7b+edT5seZdtrFUCbkEof9D9nGpahNvg8mYWf0ofx4ona4kaXm1NdPID+ljvE/dbYUX8WZRmyLjMvVQS+VxDJtsiDQIVtwbC4w+recqwDvHhLWwoeczsbEsp test@mkaas`

// CreateTestKeypair creates an SSH keypair using the V2 API client.
func CreateTestKeypair(t *testing.T, clientV2 *edgecloudV2.Client, keypairName string) (string, error) {
	t.Helper()

	opts := &edgecloudV2.KeyPairCreateRequestV2{
		SSHKeyName: keypairName,
		PublicKey:  testSSHPublicKey,
		ProjectID:  clientV2.Project,
	}

	ctx := context.Background()
	kp, _, err := clientV2.KeyPairs.CreateV2(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create keypair: %w", err)
	}

	return kp.SSHKeyID, nil
}

// DeleteTestKeypair deletes an SSH keypair using the V2 API client.
func DeleteTestKeypair(t *testing.T, clientV2 *edgecloudV2.Client, keypairID string) error {
	t.Helper()

	ctx := context.Background()
	_, err := clientV2.KeyPairs.DeleteV2(ctx, keypairID)
	if err != nil {
		return fmt.Errorf("failed to delete keypair: %w", err)
	}

	return nil
}

// --- MKaaS cluster utilities

// CreateClient creates a V2 API client for MKaaS cluster operations.
func CreateClient(t *testing.T, token, cloudAPIURL, projectID, regionID string) (*edgecloudV2.Client, error) {
	t.Helper()

	clientV2, err := edgecloudV2.NewWithRetries(nil,
		edgecloudV2.SetUserAgent("terraform-test"),
		edgecloudV2.SetAPIKey(token),
		edgecloudV2.SetBaseURL(cloudAPIURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create V2 client: %w", err)
	}

	projectIDInt, err := strconv.Atoi(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project ID: %w", err)
	}

	regionIDInt, err := strconv.Atoi(regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse region ID: %w", err)
	}

	clientV2.Project = projectIDInt
	clientV2.Region = regionIDInt

	return clientV2, nil
}

// DeleteTestMKaaSCluster deletes an MKaaS cluster using the V2 API client.
func DeleteTestMKaaSCluster(t *testing.T, clientV2 *edgecloudV2.Client, clusterID string) error {
	t.Helper()

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		return fmt.Errorf("invalid cluster id: %w", err)
	}

	ctx := context.Background()
	results, _, err := clientV2.MkaaS.ClusterDelete(ctx, clusterIDInt)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, edgecenter.MKaaSClusterDeleteTimeout)
	if err != nil {
		return fmt.Errorf("failed to wait for cluster deletion: %w", err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot delete MKaaS cluster with ID: %d", clusterIDInt)
	}

	return nil
}

// WaitForMKaaSClusterStage waits until MKaaS cluster reaches desired stage or timeout occurs.
func WaitForMKaaSClusterStage(
	t *testing.T,
	clientV2 *edgecloudV2.Client,
	clusterID string,
	desiredStage string,
	timeout time.Duration,
) error {
	t.Helper()

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		return fmt.Errorf("invalid cluster id: %w", err)
	}

	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for cluster %d to reach stage %s", clusterIDInt, desiredStage)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cluster, _, err := clientV2.MkaaS.ClusterGet(ctx, clusterIDInt)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to get cluster while waiting for stage: %w", err)
		}

		if cluster.Stage == desiredStage {
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}

func renderTemplateToWith(path, tmpl string, data any) error { //nolint:unused
	tpl := template.Must(template.New("pool").Parse(tmpl))
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	return tpl.Execute(f, data)
}

// HCL для пула + полезные outputs.
//
//nolint:unused
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
  security_group_ids = [{{- if .SecurityGroupIDs }}{{ range $i, $id := .SecurityGroupIDs }}{{ if $i }}, {{ end }}"{{ $id }}"{{ end }}{{- end }}]
  labels = {
    {{- range $k, $v := .Labels }}
    "{{ $k }}" = "{{ $v }}"
    {{- end }}
  }
}

output "pool_id"                { value = edgecenter_mkaas_pool.np.id }
output "pool_name"              { value = edgecenter_mkaas_pool.np.name }
output "out_project_id"         { value = tostring(edgecenter_mkaas_pool.np.project_id) }
output "out_region_id"          { value = tostring(edgecenter_mkaas_pool.np.region_id) }
output "out_cluster_id"         { value = tostring(edgecenter_mkaas_pool.np.cluster_id) }
output "out_flavor"             { value = edgecenter_mkaas_pool.np.flavor }
output "out_node_count"         { value = tostring(edgecenter_mkaas_pool.np.node_count) }
output "out_volume_size"        { value = tostring(edgecenter_mkaas_pool.np.volume_size) }
output "out_volume_type"        { value = edgecenter_mkaas_pool.np.volume_type }
output "out_label_env"          { value = edgecenter_mkaas_pool.np.labels["env"] }
output "out_security_group_ids" { value = edgecenter_mkaas_pool.np.security_group_ids }
output "out_state"              { value = edgecenter_mkaas_pool.np.state }
output "out_status"             { value = edgecenter_mkaas_pool.np.status }
`
