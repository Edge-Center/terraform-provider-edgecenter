//go:build integration

package mkaas_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	mkaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/mkaas/mock"
	mkaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/mkaas"
)

func controlPlane(parts map[string]interface{}) []interface{} {
	cp := map[string]interface{}{
		"flavor":      testFlavor,
		"node_count":  1,
		"volume_size": 50,
		"volume_type": string(edgecloud.VolumeTypeSsdHiIops),
		"version":     testShortVersion,
	}
	for key, value := range parts {
		cp[key] = value
	}

	return []interface{}{cp}
}

func clusterConfig(parts ...map[string]interface{}) map[string]interface{} {
	base := merge(projectRegion(), map[string]interface{}{
		"name":             testClusterName,
		"ssh_keypair_name": testKeypairName,
		"network_id":       testNetworkID,
		"subnet_id":        testSubnetID,
		"pod_subnet":       testPodSubnet,
		"service_subnet":   testServiceSubnet,
		"control_plane":    controlPlane(nil),
	})

	return merge(base, parts...)
}

func sampleCluster(parts map[string]interface{}) *edgecloud.MKaaSCluster {
	cluster := &edgecloud.MKaaSCluster{
		ID:             testClusterID,
		ProjectID:      testProjectID,
		RegionID:       testRegionID,
		Name:           testClusterName,
		SSHKeypairName: testKeypairName,
		NetworkID:      testNetworkID,
		SubnetID:       testSubnetID,
		ControlPlane: edgecloud.ControlPlane{
			Flavor:     testFlavor,
			NodeCount:  1,
			VolumeSize: 50,
			VolumeType: edgecloud.VolumeTypeSsdHiIops,
			Version:    testFullVersion,
		},
		InternalIP:    "10.0.0.10",
		ExternalIP:    "203.0.113.10",
		Created:       "2026-01-01T00:00:00Z",
		Processing:    false,
		Status:        "Provisioned",
		Stage:         "Ready",
		PodSubnet:     testPodSubnet,
		ServiceSubnet: testServiceSubnet,
	}

	if name, ok := parts["name"]; ok {
		cluster.Name = name.(string)
	}
	if version, ok := parts["version"]; ok {
		cluster.ControlPlane.Version = version.(string)
	}
	if nodeCount, ok := parts["node_count"]; ok {
		cluster.ControlPlane.NodeCount = nodeCount.(int)
	}

	return cluster
}

func clusterState(parts ...map[string]interface{}) map[string]interface{} {
	return merge(clusterConfig(), parts...)
}

func TestIntegrationMKaaSClusterResource_Create(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	var sent edgecloud.MKaaSClusterCreateRequest

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name: "create resolves the short version through the api, sends the whole config and stores the id",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions("v1.30.9", testFullVersion), nil, nil)
				mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(1).(edgecloud.MKaaSClusterCreateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(clusterCreated(testClusterID)), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(nil), nil, nil).Once()

				return mc
			},
			NewConfig: clusterConfig(map[string]interface{}{"publish_kube_api_to_internet": true}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)

				require.Equal(t, testClusterName, sent.Name)
				require.Equal(t, testKeypairName, sent.SSHKeyPairName)
				require.Equal(t, testNetworkID, sent.NetworkID)
				require.Equal(t, testSubnetID, sent.SubnetID)
				require.True(t, sent.PublishKubeAPIToInternet)
				require.NotNil(t, sent.PodSubnet)
				require.Equal(t, testPodSubnet, *sent.PodSubnet)
				require.NotNil(t, sent.ServiceSubnet)
				require.Equal(t, testServiceSubnet, *sent.ServiceSubnet)
				require.Equal(t, edgecloud.ControlPlaneCreateRequest{
					Flavor:     testFlavor,
					NodeCount:  1,
					VolumeSize: 50,
					VolumeType: edgecloud.VolumeTypeSsdHiIops,
					Version:    testFullVersion,
				}, sent.ControlPlane)

				support.RequireStateAttrs(t, state, map[string]string{
					"name":                       testClusterName,
					"ssh_keypair_name":           testKeypairName,
					"network_id":                 testNetworkID,
					"subnet_id":                  testSubnetID,
					"pod_subnet":                 testPodSubnet,
					"service_subnet":             testServiceSubnet,
					"project_id":                 testProjectIDStr,
					"region_id":                  testRegionIDStr,
					"control_plane.0.flavor":     testFlavor,
					"control_plane.0.node_count": "1",
					"control_plane.0.version":    testShortVersion,
					"internal_ip":                "10.0.0.10",
					"external_ip":                "203.0.113.10",
					"created":                    "2026-01-01T00:00:00Z",
					"processing":                 "false",
					"status":                     "Provisioned",
					"stage":                      "Ready",
				})
			},
		},
		{
			Name: "create picks the lexicographically last patch instead of the newest one",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions("v1.31.4", "v1.31.10"), nil, nil)
				mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent = args.Get(1).(edgecloud.MKaaSClusterCreateRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(clusterCreated(testClusterID)), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(nil), nil, nil).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "v1.31.4", sent.ControlPlane.Version,
					"v1.31.10 is newer, sort.Strings puts v1.31.4 last")
			},
		},
		{
			Name: "an unavailable minor version aborts before the create call",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions("v1.29.2", "v1.30.9"), nil, nil)

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags,
					`control_plane.version "v1.31" is not available in region 8; available versions: v1.29, v1.30`)
				require.Nil(t, state)
				mc.MKaaS.AssertNotCalled(t, "ClusterCreate", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "a fully qualified version is rejected although the api offers it",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions("v1.30.9", testFullVersion), nil, nil)

				return mc
			},
			NewConfig: clusterConfig(map[string]interface{}{
				"control_plane": controlPlane(map[string]interface{}{"version": testFullVersion}),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags,
					`control_plane.version "v1.31.4" is not available in region 8`)
				mc.MKaaS.AssertNotCalled(t, "ClusterCreate", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "an empty version list produces the dedicated message",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(), nil, nil)

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "the API returned no available versions")
			},
		},
		{
			Name: "a versions api failure is wrapped with the region",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(nil, statusResponse(500), errors.New("backend is down"))

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags,
					"failed to fetch available k8s versions for region 8: backend is down")
			},
		},
		{
			Name: "a create api error surfaces and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(testFullVersion), nil, nil)
				mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(nil, statusResponse(409), errors.New("cluster with that name already exists")).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "error from creating mkaas")
				require.Nil(t, state)
			},
		},
		{
			Name: "a create task that ends in error orphans the cluster",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(testFullVersion), nil, nil)
				mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				require.Nil(t, state, "the cluster the task started is not recorded anywhere")
				mc.MKaaS.AssertNotCalled(t, "ClusterGet", mock.Anything, mock.Anything)
			},
		},
		{
			Name: "a 404 from the trailing read turns a successful create into a silent no-op",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(testFullVersion), nil, nil)
				mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(clusterCreated(testClusterID)), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), errors.New("cluster not found")).Once()

				return mc
			},
			NewConfig: clusterConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state, "the cluster exists in the cloud and nothing records it")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSClusterResource_CreatePanics(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	t.Run("a create answer without task ids crashes the provider", func(t *testing.T) {
		t.Parallel()

		mc := mkaasmock.NewMockedMKaaS()
		mkaasmock.AllowProjectResolution(mc, testProjectID)
		mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
			Return(availableVersions(testFullVersion), nil, nil)
		mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
			Return(taskResponse(), nil, nil).Once()

		require.Panics(t, func() {
			support.ApplyConfig(t, t.Context(), resource, nil, clusterConfig(), mc.TestMeta())
		})
	})

	t.Run("a create task that carries no cluster id crashes the provider", func(t *testing.T) {
		t.Parallel()

		mc := mkaasmock.NewMockedMKaaS()
		mkaasmock.AllowProjectResolution(mc, testProjectID)
		mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
			Return(availableVersions(testFullVersion), nil, nil)
		mc.MKaaS.On("ClusterCreate", mock.Anything, mock.Anything).
			Return(taskResponse(testTaskID), nil, nil).Once()
		mc.Tasks.On("Get", mock.Anything, testTaskID).
			Return(finishedTask(nil), nil, nil).Once()

		require.Panics(t, func() {
			support.ApplyConfig(t, t.Context(), resource, nil, clusterConfig(), mc.TestMeta())
		})
	})
}

func TestIntegrationMKaaSClusterResource_Read(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "read maps every attribute back and shortens the control plane version",
			Op:        support.OpRead,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(map[string]interface{}{"name": "renamed-out-of-band"}), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                    "renamed-out-of-band",
					"control_plane.0.version": testShortVersion,
					"project_id":              testProjectIDStr,
					"region_id":               testRegionIDStr,
					"status":                  "Provisioned",
					"stage":                   "Ready",
				})
			},
		},
		{
			Name:      "read drops the resource when the cluster is gone",
			Op:        support.OpRead,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, notFound(), errors.New("cluster not found")).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name:      "read keeps the resource and reports a non 404 failure",
			Op:        support.OpRead,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(nil, statusResponse(500), errors.New("internal error")).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)
			},
		},
		{
			Name:      "read never writes publish_kube_api_to_internet back",
			Op:        support.OpRead,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(nil), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(map[string]interface{}{"publish_kube_api_to_internet": true}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "true", state.Attributes["publish_kube_api_to_internet"],
					"read leaves whatever was already in state, the api never reports the flag")
			},
		},
		{
			Name:         "read rejects a non numeric id without touching the api",
			Op:           support.OpRead,
			CurrentID:    "not-a-number",
			CurrentState: clusterState(),
			Prepare: func() *mkaasmock.MockedMKaaS {
				return mkaasmock.NewMockedMKaaS()
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "invalid cluster id")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSClusterResource_ReadPanicsWithoutAResponse(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	mc := mkaasmock.NewMockedMKaaS()
	mkaasmock.AllowProjectResolution(mc, testProjectID)
	mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
		Return(nil, (*edgecloud.Response)(nil), errors.New("request could not be built")).Once()

	state := support.NewState(t, resource, clusterState(), testClusterIDStr)
	data := support.NewResourceDataFromState(t, resource, state)

	require.Panics(t, func() {
		resource.ReadContext(t.Context(), data, mc.TestMeta())
	})
}

func TestIntegrationMKaaSClusterResource_Update(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "an unsupported change is rejected before any api call",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				return mkaasmock.NewMockedMKaaS()
			},
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"ssh_keypair_name": "other-key"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags,
					"MKaaS cluster update is not supported for these fields: [ssh_keypair_name]")
				require.Equal(t, "other-key", state.Attributes["ssh_keypair_name"],
					"the refused value is still written into state")
			},
		},
		{
			Name:      "a rename sends the name and waits for its task",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterUpdateName", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpdateNameRequest{Name: "renamed"}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(map[string]interface{}{"name": "renamed"}), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"name": "renamed"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"name": "renamed"})
			},
		},
		{
			Name:      "a control plane node count change sends the master node count",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterUpdateMasterNodeCount", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpdateMasterNodeCountRequest{MasterNodeCount: 3}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(map[string]interface{}{"node_count": 3}), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			NewConfig: clusterConfig(map[string]interface{}{
				"control_plane": controlPlane(map[string]interface{}{"node_count": 3}),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"control_plane.0.node_count": "3"})
			},
		},
		{
			Name:      "a version bump resolves the target through the api first",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(testFullVersion, "v1.32.1"), nil, nil).Once()
				mc.MKaaS.On("ClusterUpgradeVersion", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpgradeVersionRequest{TargetVersion: "v1.32.1"}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(map[string]interface{}{"version": "v1.32.1"}), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			NewConfig: clusterConfig(map[string]interface{}{
				"control_plane": controlPlane(map[string]interface{}{"version": "v1.32"}),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"control_plane.0.version": "v1.32"})
			},
		},
		{
			Name:      "name, node count and version change together in one apply",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterUpdateName", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpdateNameRequest{Name: "renamed"}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.MKaaS.On("ClusterUpdateMasterNodeCount", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpdateMasterNodeCountRequest{MasterNodeCount: 3}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.MKaaS.On("VersionsList", mock.Anything, testRegionID).
					Return(availableVersions(testFullVersion, "v1.32.1"), nil, nil).Once()
				mc.MKaaS.On("ClusterUpgradeVersion", mock.Anything, testClusterID,
					edgecloud.MKaaSClusterUpgradeVersionRequest{TargetVersion: "v1.32.1"}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Times(3)
				mc.MKaaS.On("ClusterGet", mock.Anything, testClusterID).
					Return(sampleCluster(map[string]interface{}{
						"name":       "renamed",
						"node_count": 3,
						"version":    "v1.32.1",
					}), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			NewConfig: clusterConfig(map[string]interface{}{
				"name": "renamed",
				"control_plane": controlPlane(map[string]interface{}{
					"node_count": 3,
					"version":    "v1.32",
				}),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                       "renamed",
					"control_plane.0.node_count": "3",
					"control_plane.0.version":    "v1.32",
				})
			},
		},
		{
			Name:      "an update whose task fails aborts before the read",
			Op:        support.OpApply,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterUpdateName", mock.Anything, testClusterID, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			NewConfig:    clusterConfig(map[string]interface{}{"name": "renamed"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				mc.MKaaS.AssertNotCalled(t, "ClusterGet", mock.Anything, mock.Anything)
				require.Equal(t, "renamed", state.Attributes["name"],
					"the name the api never accepted is recorded anyway")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSClusterResource_Delete(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSClusterResource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "delete waits for the task and clears the id",
			Op:        support.OpDelete,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name:      "deleting an already deleted cluster fails and keeps it in state",
			Op:        support.OpDelete,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(nil, notFound(), errors.New("cluster not found")).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testClusterIDStr)
			},
		},
		{
			Name:      "a delete task in the error state reports the sdk wording, never the resource wording",
			Op:        support.OpDelete,
			CurrentID: testClusterIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("ClusterDelete", mock.Anything, testClusterID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				for _, d := range diags {
					require.NotContains(t, d.Summary, "cannot delete MKaaS cluster with ID",
						"the TaskStateError branch below the wait is unreachable")
				}
				support.RequireStateID(t, state, testClusterIDStr)
			},
		},
		{
			Name:      "delete silently forgets a cluster whose id is not a number",
			Op:        support.OpDelete,
			CurrentID: "not-a-number",
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				return mc
			},
			CurrentState: clusterState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireNoDiags(t, diags)
				require.Nil(t, state, "terraform records a successful destroy and the cluster is leaked")
				mc.MKaaS.AssertNotCalled(t, "ClusterDelete", mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}
