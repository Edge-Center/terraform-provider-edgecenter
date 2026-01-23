//go:build cloud_data_source_mkaas

package edgecenter_test

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/random"
	tt "github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const mkaasClusterDSTmpl = `
terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}

provider "edgecenter" {
  permanent_api_token = "{{ .Token }}"
}

data "edgecenter_mkaas_cluster" "ds" {
  project_id = {{ .ProjectID }}
  region_id  = {{ .RegionID }}
  id         = "{{ .ClusterID }}"
}

output "id"               { value = data.edgecenter_mkaas_cluster.ds.id }
output "name"             { value = data.edgecenter_mkaas_cluster.ds.name }
output "out_project_id"   { value = tostring(data.edgecenter_mkaas_cluster.ds.project_id) }
output "out_region_id"    { value = tostring(data.edgecenter_mkaas_cluster.ds.region_id) }
output "out_ssh_keypair"  { value = data.edgecenter_mkaas_cluster.ds.ssh_keypair_name }
output "out_network_id"   { value = data.edgecenter_mkaas_cluster.ds.network_id }
output "out_subnet_id"    { value = data.edgecenter_mkaas_cluster.ds.subnet_id }

output "out_cp_flavor"       { value = data.edgecenter_mkaas_cluster.ds.control_plane[0].flavor }
output "out_cp_node_count"   { value = tostring(data.edgecenter_mkaas_cluster.ds.control_plane[0].node_count) }
output "out_cp_volume_size"  { value = tostring(data.edgecenter_mkaas_cluster.ds.control_plane[0].volume_size) }
output "out_cp_volume_type"  { value = data.edgecenter_mkaas_cluster.ds.control_plane[0].volume_type }
output "out_cp_version"      { value = data.edgecenter_mkaas_cluster.ds.control_plane[0].version }

output "out_internal_ip"  { value = data.edgecenter_mkaas_cluster.ds.internal_ip }
output "out_external_ip"  { value = data.edgecenter_mkaas_cluster.ds.external_ip }
output "out_created"      { value = data.edgecenter_mkaas_cluster.ds.created }
output "out_processing"   { value = tostring(data.edgecenter_mkaas_cluster.ds.processing) }
output "out_status"       { value = data.edgecenter_mkaas_cluster.ds.status }
output "out_stage"        { value = data.edgecenter_mkaas_cluster.ds.stage }
`

type mkaasClusterDSData struct {
	Token     string
	ProjectID string
	RegionID  string
	ClusterID string
}

func TestMKaaSClusterDataSource_ReadByID(t *testing.T) {

	t.Log("Starting TestMKaaSClusterDataSource_ReadByID")

	token := requireEnv(t, "EC_PERMANENT_TOKEN")
	cloudAPIURL := requireEnv(t, "EC_API")
	projectID := requireEnv(t, "TEST_PROJECT_ID")
	regionID := requireEnv(t, "TEST_MKAAS_REGION_ID")

	cpFlavor := "g3-standard-2-4"

	volType := "ssd_hiiops"

	k8sVersion := "v1.31.0"

	base := "tf-mkaas-ds-" + strings.ToLower(random.UniqueId())

	client, err := CreateClient(t, token, cloudAPIURL, projectID, regionID)
	require.NoError(t, err, "failed to create client")

	keypairName := base + "-key"
	keypairID, err := CreateTestKeypair(t, client, keypairName)
	require.NoError(t, err, "failed to create SSH keypair")
	t.Cleanup(func() {
		if err := DeleteTestKeypair(t, client, keypairID); err != nil {
			t.Errorf("cleanup failed: delete SSH keypair %s: %v", keypairID, err)
		}
	})

	networkName := base + "-net"
	networkID, err := CreateTestNetwork(client, &edgecloudV2.NetworkCreateRequest{
		Name:         networkName,
		Type:         edgecloudV2.VXLAN,
		CreateRouter: true,
	})
	require.NoError(t, err, "failed to create network")
	t.Cleanup(func() {
		if err := DeleteTestNetwork(client, networkID); err != nil {
			t.Errorf("cleanup failed: delete network %s: %v", networkID, err)
		}
	})

	subnetName := base + "-subnet"
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

	clusterName := base + "-cls"
	cluster, err := CreateCluster(t, tfData{
		Token:                    token,
		ProjectID:                projectID,
		RegionID:                 regionID,
		NetworkID:                networkID,
		SubnetID:                 subnetID,
		SSHKeypair:               keypairName,
		PodSubnet:                podSubnet,
		ServiceSubnet:            serviceSubnet,
		PublishKubeApiToInternet: false,
		Name:                     clusterName,
		CPFlavor:                 cpFlavor,
		CPNodeCount:              1,
		CPVolumeSize:             30,
		CPVolumeType:             volType,
		CPVersion:                k8sVersion,
	})
	require.NoError(t, err, "failed to create cluster")
	require.NotEmpty(t, cluster.ID, "cluster ID is empty")
	var testSucceed bool
	t.Cleanup(func() {
		if cluster != nil && !testSucceed {
			if err := DeleteTestMKaaSCluster(t, client, cluster.ID); err != nil {
				t.Errorf("cleanup failed: delete cluster %s via API: %v", cluster.ID, err)
			}
		}
	})

	dsDir := filepath.Join(cluster.Dir, "datasource")
	require.NoError(t, os.MkdirAll(dsDir, 0755), "mkdir datasource dir")

	dsMain := filepath.Join(dsDir, "main.tf")
	dsData := mkaasClusterDSData{
		Token:     token,
		ProjectID: projectID,
		RegionID:  regionID,
		ClusterID: cluster.ID,
	}
	require.NoError(t, renderTemplateToWith(dsMain, mkaasClusterDSTmpl, dsData), "write datasource main.tf")

	dsOpts := &tt.Options{
		TerraformDir: dsDir,
		NoColor:      true,
	}

	if _, err := tt.ApplyAndIdempotentE(t, dsOpts); err != nil {
		t.Fatalf("terraform apply (datasource): %v", err)
	}

	require.Equal(t, cluster.ID, tt.Output(t, dsOpts, "id"))
	require.Equal(t, clusterName, tt.Output(t, dsOpts, "name"))

	require.Equal(t, projectID, tt.Output(t, dsOpts, "out_project_id"))
	require.Equal(t, regionID, tt.Output(t, dsOpts, "out_region_id"))

	require.Equal(t, keypairName, tt.Output(t, dsOpts, "out_ssh_keypair"))
	require.Equal(t, networkID, tt.Output(t, dsOpts, "out_network_id"))
	require.Equal(t, subnetID, tt.Output(t, dsOpts, "out_subnet_id"))

	require.Equal(t, cpFlavor, tt.Output(t, dsOpts, "out_cp_flavor"))
	require.Equal(t, "1", tt.Output(t, dsOpts, "out_cp_node_count"))
	require.Equal(t, "30", tt.Output(t, dsOpts, "out_cp_volume_size"))
	require.Equal(t, volType, tt.Output(t, dsOpts, "out_cp_volume_type"))
	require.Equal(t, k8sVersion, tt.Output(t, dsOpts, "out_cp_version"))

	_ = tt.Output(t, dsOpts, "out_internal_ip")
	_ = tt.Output(t, dsOpts, "out_external_ip")
	require.NotEmpty(t, tt.Output(t, dsOpts, "out_created"))
	require.NotEmpty(t, tt.Output(t, dsOpts, "out_status"))
	require.Equal(t, clusterWorkCompletedStage, tt.Output(t, dsOpts, "out_stage"))

	if err := cluster.Destroy(t); err != nil {
		t.Fatalf("terraform destroy for cluster: %v", err)
	}
	testSucceed = true
}
