package edgecenter_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"

	tt "github.com/gruntwork-io/terratest/modules/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	ec "github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
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

// --- network and subnet utilities

// CreateNetworkAndSubnetClients creates API clients for network and subnet operations
func CreateNetworkAndSubnetClients(t *testing.T, token, endpoint, projectID, regionID string) (*edgecloud.ServiceClient, *edgecloud.ServiceClient, error) {
	t.Helper()

	cloudAPI := endpoint
	if !strings.HasSuffix(cloudAPI, "/cloud") {
		cloudAPI = endpoint + "/cloud"
	}

	t.Logf("Creating provider client with API URL: %s", cloudAPI)
	provider, err := ec.APITokenClient(edgecloud.APITokenOptions{
		APIURL:   cloudAPI,
		APIToken: token,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create provider client: %w", err)
	}
	t.Log("Provider client created successfully")

	projectIDInt, err := strconv.Atoi(projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse project ID: %w", err)
	}

	regionIDInt, err := strconv.Atoi(regionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse region ID: %w", err)
	}

	networkClient, err := ec.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    edgecenter.NetworksPoint,
		Region:  regionIDInt,
		Project: projectIDInt,
		Version: edgecenter.VersionPointV1,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create network client: %w", err)
	}

	subnetClient, err := ec.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    edgecenter.SubnetPoint,
		Region:  regionIDInt,
		Project: projectIDInt,
		Version: edgecenter.VersionPointV1,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create subnet client: %w", err)
	}

	return networkClient, subnetClient, nil
}

// CreateTestNetwork creates a network using the API client
func CreateTestNetwork(client *edgecloud.ServiceClient, opts networks.CreateOpts) (string, error) {
	// Note: t.Log is not available here, so we can't add logging
	result, err := networks.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	timeoutSeconds := int(edgecenter.NetworkCreatingTimeout.Seconds())
	networkID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, timeoutSeconds, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		networkID, err := networks.ExtractNetworkIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve network ID from task info: %w", err)
		}
		return networkID, nil
	})
	if err != nil {
		return "", err
	}

	return networkID.(string), nil
}

// DeleteTestNetwork deletes a network using the API client
func DeleteTestNetwork(client *edgecloud.ServiceClient, networkID string) error {
	result, err := networks.Delete(client, networkID).Extract()
	if err != nil {
		return err
	}

	taskID := result.Tasks[0]
	err = tasks.WaitTaskAndProcessResult(client, taskID, true, int(edgecenter.NetworkDeletingTimeout.Seconds()), func(task tasks.TaskID) error {
		_, err := networks.Get(client, networkID).Extract()
		if err == nil {
			return fmt.Errorf("cannot delete network with ID: %s", networkID)
		}

		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil
		}
		return fmt.Errorf("extracting Network resource error: %w", err)
	})

	return err
}

// CreateTestSubnet creates a subnet using the API client
func CreateTestSubnet(client *edgecloud.ServiceClient, opts subnets.CreateOpts, cidr string) (string, error) {
	var eccidr edgecloud.CIDR
	_, netIPNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	eccidr.IP = netIPNet.IP
	eccidr.Mask = netIPNet.Mask
	opts.CIDR = eccidr

	result, err := subnets.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := result.Tasks[0]
	subnetID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, int(edgecenter.SubnetCreatingTimeout.Seconds()), func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		subnet, err := subnets.ExtractSubnetIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve Subnet ID from task info: %w", err)
		}
		return subnet, nil
	})

	return subnetID.(string), err
}

// DeleteTestSubnet deletes a subnet using the API client
func DeleteTestSubnet(client *edgecloud.ServiceClient, subnetID string) error {
	result, err := subnets.Delete(client, subnetID).Extract()
	if err != nil {
		return err
	}

	taskID := result.Tasks[0]
	err = tasks.WaitTaskAndProcessResult(client, taskID, true, int(edgecenter.SubnetCreatingTimeout.Seconds()), func(task tasks.TaskID) error {
		_, err := subnets.Get(client, subnetID).Extract()
		if err == nil {
			return fmt.Errorf("cannot delete subnet with ID: %s", subnetID)
		}

		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil
		}
		return fmt.Errorf("extracting Subnet resource error: %w", err)
	})

	return err
}

// --- SSH keypair utilities

// Test SSH public key for dynamic keypair creation
const testSSHPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC1bdbQYquD/swsZpFPXagY9KvhlNUTKYMdhRNtlGglAMgRxJS3Q0V74BNElJtP+UU/AbZD4H2ZAwW3PLLD/maclnLlrA48xg/ez9IhppBop0WADZ/nB4EcvQfR/Db7nHDTZERW6EiiGhV6CkHVasK2sY/WNRXqPveeWUlwCqtSnU90l/s9kQCoEfkM2auO6ppJkVrXbs26vcRclS8KL7Cff4HwdVpV7b+edT5seZdtrFUCbkEof9D9nGpahNvg8mYWf0ofx4ona4kaXm1NdPID+ljvE/dbYUX8WZRmyLjMvVQS+VxDJtsiDQIVtwbC4w+recqwDvHhLWwoeczsbEsp test@mkaas`

// CreateKeypairClient creates a V2 API client for keypair operations
func CreateKeypairClient(t *testing.T, token, endpoint, projectID string) (*edgecloudV2.Client, error) {
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

	// For KeyPairsV2 endpoints, only project is needed, region is set to stub value 1
	clientV2.Project = projectIDInt
	clientV2.Region = 1

	return clientV2, nil
}

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

// CreateMKaaSClient creates a V2 API client for MKaaS cluster operations
func CreateMKaaSClient(t *testing.T, token, endpoint, projectID, regionID string) (*edgecloudV2.Client, error) {
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
