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

	tt "github.com/gruntwork-io/terratest/modules/terraform"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
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
	once     sync.Once
}

// CreateCluster
func CreateCluster(t *testing.T, data tfData) (*Cluster, func() error, error) {
	t.Helper()

	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.tf")

	fileMainTFCloser, err := renderTemplateTo(mainPath, data)
	if err != nil {
		return nil, nil, fmt.Errorf("write main.tf: %w", err)
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

	if _, err := tt.ApplyAndIdempotentE(t, opts); err != nil {
		t.Logf("terraform apply failed: %v", err)
		if _, destroyErr := tt.DestroyE(t, opts); destroyErr != nil {
			t.Logf("terraform destroy attempt after failed apply returned error: %v", destroyErr)
		}
		return nil, fileMainTFCloser, fmt.Errorf("terraform apply: %w", err)
	}

	id, err := tt.OutputRequiredE(t, opts, "cluster_id")
	if err != nil || strings.TrimSpace(id) == "" {
		if _, destroyErr := tt.DestroyE(t, opts); destroyErr != nil {
			t.Logf("terraform destroy attempt after empty cluster_id returned error: %v", destroyErr)
		}
		if err != nil {
			return nil, fileMainTFCloser, fmt.Errorf("cluster_id output: %w", err)
		}
		return nil, fileMainTFCloser, fmt.Errorf("cluster_id is empty after create")
	}

	c := &Cluster{
		Dir:      tmp,
		MainPath: mainPath,
		Opts:     opts,
		Data:     data,
		ID:       id,
	}
	return c, fileMainTFCloser, nil
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

// UpdateCluster
func (c *Cluster) UpdateCluster(t *testing.T, mutate func(*tfData)) (func() error, error) {
	t.Helper()
	if mutate != nil {
		mutate(&c.Data)
	}
	fileMainTFCloser, err := renderTemplateTo(c.MainPath, c.Data)
	if err != nil {
		return fileMainTFCloser, fmt.Errorf("write main.tf (update): %w", err)
	}

	_, err = tt.ApplyAndIdempotentE(t, c.Opts)
	if err != nil {
		return fileMainTFCloser, fmt.Errorf("terraform apply (update): %w", err)
	}
	return fileMainTFCloser, nil
}

// ImportClusterPlanApply
func ImportClusterPlanApply(t *testing.T, token, endpoint, projectID, regionID, clusterID, workDir string, retry map[string]string) (*tt.Options, error) {
	t.Helper()

	importDir := filepath.Join(workDir, "import")
	if err := os.MkdirAll(importDir, 0755); err != nil {
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
	if err := os.WriteFile(filepath.Join(importDir, "main.tf"), []byte(importMain), 0644); err != nil {
		return nil, fmt.Errorf("write import/main.tf: %w", err)
	}

	opts := &tt.Options{
		TerraformDir:             importDir,
		NoColor:                  true,
		RetryableTerraformErrors: retry,
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
		return nil, fmt.Errorf("terraform plan after import/apply is not empty (err=%v)\n%s", err, out)
	}

	return opts, nil
}

// --- common utils

func renderTemplateTo(path string, data tfData) (func() error, error) {
	tpl := template.Must(template.New("main").Parse(mainTmpl))
	f, err := os.Create(path)
	if err != nil {
		return f.Close, err
	}
	return f.Close, tpl.Execute(f, data)
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Skipf("missing %s; skipping integration test", key)
	}
	return val
}

// CreateTestNetwork создаёт сеть через V2 API
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

// DeleteTestNetwork удаляет сеть через V2 API
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

// CreateTestSubnet создаёт подсеть через V2 API
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

// DeleteTestSubnet удаляет подсеть через V2 API
func DeleteTestSubnet(client *edgecloudV2.Client, subnetID string) error {
	ctx := context.Background()

	results, _, err := client.Subnetworks.Delete(ctx, subnetID)
	if err != nil {
		return err
	}

	if len(results.Tasks) == 0 {
		return fmt.Errorf("no task returned for subnet deletion")
	}
	taskID := results.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, client, taskID, edgecenter.SubnetCreatingTimeout)
	if err != nil {
		return err
	}

	if taskInfo.State == edgecloudV2.TaskStateError {
		return fmt.Errorf("cannot delete subnet with ID: %s", subnetID)
	}

	return nil
}

// --- SSH keypair utilities

// Test SSH public key for dynamic keypair creation
const testSSHPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC1bdbQYquD/swsZpFPXagY9KvhlNUTKYMdhRNtlGglAMgRxJS3Q0V74BNElJtP+UU/AbZD4H2ZAwW3PLLD/maclnLlrA48xg/ez9IhppBop0WADZ/nB4EcvQfR/Db7nHDTZERW6EiiGhV6CkHVasK2sY/WNRXqPveeWUlwCqtSnU90l/s9kQCoEfkM2auO6ppJkVrXbs26vcRclS8KL7Cff4HwdVpV7b+edT5seZdtrFUCbkEof9D9nGpahNvg8mYWf0ofx4ona4kaXm1NdPID+ljvE/dbYUX8WZRmyLjMvVQS+VxDJtsiDQIVtwbC4w+recqwDvHhLWwoeczsbEsp test@mkaas`

// CreateTestKeypair creates an SSH keypair using the V2 API client
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

// DeleteTestKeypair deletes an SSH keypair using the V2 API client
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

// CreateClient creates a V2 API client for MKaaS cluster operations
func CreateClient(t *testing.T, token, endpoint, projectID, regionID string) (*edgecloudV2.Client, error) {
	t.Helper()

	cloudAPI := endpoint
	if !strings.HasSuffix(cloudAPI, "/cloud") {
		cloudAPI = endpoint + "/cloud"
	}

	clientV2, err := edgecloudV2.NewWithRetries(nil,
		edgecloudV2.SetUserAgent("terraform-test"),
		edgecloudV2.SetAPIKey(token),
		edgecloudV2.SetBaseURL(cloudAPI),
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

// DeleteTestMKaaSCluster deletes an MKaaS cluster using the V2 API client
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

// --- cleanup helpers

type MSaaSTestCleaner struct {
	t         *testing.T
	client    *edgecloudV2.Client
	cluster   *Cluster
	networkID string
	subnetID  string
	keypairID string
	once      sync.Once
	// fifo 0->1->2
	cleaners []func() error
}

func NewMSaaSTestCleaner(t *testing.T, client *edgecloudV2.Client) *MSaaSTestCleaner {
	c := &MSaaSTestCleaner{
		t:      t,
		client: client,
	}
	c.AddCleaner(c.cleanupCluster)
	c.AddCleaner(c.cleanupSubnet)
	c.AddCleaner(c.cleanupNetwork)
	c.AddCleaner(c.cleanupKeypair)
	t.Cleanup(c.Run)
	return c
}

func (c *MSaaSTestCleaner) AddCleaner(cleaner func() error) {
	c.cleaners = append(c.cleaners, cleaner)
}

func (c *MSaaSTestCleaner) AttachCluster(cluster *Cluster) {
	c.cluster = cluster
}

func (c *MSaaSTestCleaner) SetNetworkID(id string) {
	c.networkID = id
}

func (c *MSaaSTestCleaner) SetSubnetID(id string) {
	c.subnetID = id
}

func (c *MSaaSTestCleaner) SetKeypairID(id string) {
	c.keypairID = id
}

func (c *MSaaSTestCleaner) Run() {
	c.once.Do(func() {
		for _, cleaner := range c.cleaners {
			err := cleaner()
			if err != nil {
				c.t.Fatalf("filed to clean resource:%v", err)
			}
		}
	})
}

func (c *MSaaSTestCleaner) cleanupCluster() error {
	if c.cluster != nil {
		if err := c.cluster.Destroy(c.t); err != nil {
			c.t.Logf("terraform destroy (cluster) failed: %v", err)
			if err := DeleteTestMKaaSCluster(c.t, c.client, c.cluster.ID); err != nil {
				return fmt.Errorf("failed to delete cluster %s via API: %v", c.cluster.ID, err)
			}
		}
		c.cluster = nil
	}
	return nil
}

func (c *MSaaSTestCleaner) cleanupNetwork() error {
	if c.networkID == "" {
		return nil
	}

	if err := DeleteTestNetwork(c.client, c.networkID); err != nil {
		return fmt.Errorf("failed to delete network %s: %v", c.networkID, err)
	}
	c.networkID = ""

	return nil
}

func (c *MSaaSTestCleaner) cleanupSubnet() error {
	if c.subnetID == "" {
		return nil
	}
	if err := DeleteTestSubnet(c.client, c.subnetID); err != nil {
		return fmt.Errorf("failed to delete subnet %s: %v", c.subnetID, err)
	}
	c.subnetID = ""

	return nil
}

func (c *MSaaSTestCleaner) cleanupKeypair() error {
	if c.keypairID == "" {
		return nil
	}
	if err := DeleteTestKeypair(c.t, c.client, c.keypairID); err != nil {
		return fmt.Errorf("failed to delete SSH keypair %s: %v", c.keypairID, err)
	}

	c.keypairID = ""
	return nil
}
