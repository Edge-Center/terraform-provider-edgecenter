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

func scalePolicy(minNodeCount, maxNodeCount int) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"auto_scale": []interface{}{
				map[string]interface{}{
					"min_node_count": minNodeCount,
					"max_node_count": maxNodeCount,
				},
			},
		},
	}
}

func taint(key, value, effect string) map[string]interface{} {
	return map[string]interface{}{"key": key, "value": value, "effect": effect}
}

func poolConfig(parts ...map[string]interface{}) map[string]interface{} {
	base := merge(projectRegion(), map[string]interface{}{
		"cluster_id":  testClusterID,
		"name":        testPoolName,
		"flavor":      testFlavor,
		"node_count":  2,
		"volume_size": 20,
		"volume_type": string(edgecloud.VolumeTypeStandard),
	})

	return merge(base, parts...)
}

func samplePool(parts map[string]interface{}) *edgecloud.MKaaSPool {
	pool := &edgecloud.MKaaSPool{
		ID:         testPoolID,
		Name:       testPoolName,
		Flavor:     testFlavor,
		NodeCount:  2,
		VolumeSize: 20,
		VolumeType: edgecloud.VolumeTypeStandard,
		State:      "ACTIVE",
		Status:     "Provisioned",
	}

	for key, value := range parts {
		switch key {
		case "name":
			pool.Name = value.(string)
		case "node_count":
			pool.NodeCount = value.(int)
		case "autoscaling_enabled":
			pool.AutoscalingEnabled = value.(bool)
		case "min_node_count":
			pool.MinNodeCount = value.(int)
		case "max_node_count":
			pool.MaxNodeCount = value.(int)
		case "labels":
			pool.Labels = value.(map[string]string)
		case "taints":
			pool.Taints = value.([]edgecloud.MKaaSTaint)
		case "security_group_ids":
			pool.SecurityGroupIds = value.([]string)
		}
	}

	return pool
}

func poolState(parts ...map[string]interface{}) map[string]interface{} {
	return merge(poolConfig(), parts...)
}

func TestIntegrationMKaaSPoolResource_Create(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	var sent edgecloud.MKaaSPoolCreateRequest

	capture := func(args mock.Arguments) {
		sent = args.Get(2).(edgecloud.MKaaSPoolCreateRequest)
	}

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name: "create a fixed size pool sends the whole config and stores the id",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Run(capture).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(poolCreated(testPoolID)), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(nil), nil, nil).Once()

				return mc
			},
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testPoolIDStr)

				require.Equal(t, testPoolName, sent.Name)
				require.Equal(t, testFlavor, sent.Flavor)
				require.Equal(t, 2, sent.NodeCount)
				require.Equal(t, 20, sent.VolumeSize)
				require.Equal(t, edgecloud.VolumeTypeStandard, sent.VolumeType)
				require.Contains(t, marshal(t, sent), `"node_count":2`)
				require.False(t, sent.AutoscalingEnabled)
				require.Nil(t, sent.MinNodeCount)
				require.Nil(t, sent.MaxNodeCount)
				require.Empty(t, sent.Labels)
				require.Empty(t, sent.Taints)

				support.RequireStateAttrs(t, state, map[string]string{
					"cluster_id":         testClusterIDStr,
					"name":               testPoolName,
					"flavor":             testFlavor,
					"node_count":         "2",
					"current_node_count": "2",
					"volume_size":        "20",
					"volume_type":        string(edgecloud.VolumeTypeStandard),
					"state":              "ACTIVE",
					"status":             "Provisioned",
					"project_id":         testProjectIDStr,
					"region_id":          testRegionIDStr,
				})
			},
		},
		{
			Name: "create with autoscaling seeds the node count from the minimum",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Run(capture).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(poolCreated(testPoolID)), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"autoscaling_enabled": true,
						"min_node_count":      2,
						"max_node_count":      5,
						"node_count":          4,
					}), nil, nil).Once()

				return mc
			},
			NewConfig: merge(poolConfig(), map[string]interface{}{
				"node_count":   nil,
				"scale_policy": scalePolicy(2, 5),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)

				require.True(t, sent.AutoscalingEnabled)
				require.NotNil(t, sent.MinNodeCount)
				require.Equal(t, 2, *sent.MinNodeCount)
				require.NotNil(t, sent.MaxNodeCount)
				require.Equal(t, 5, *sent.MaxNodeCount)
				require.Equal(t, 2, sent.NodeCount, "the node count is seeded from min_node_count")

				support.RequireStateAttrs(t, state, map[string]string{
					"scale_policy.0.auto_scale.0.min_node_count": "2",
					"scale_policy.0.auto_scale.0.max_node_count": "5",
					"current_node_count":                         "4",
				})
			},
		},
		{
			Name: "create carries labels, taints and security group ids",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Run(capture).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(poolCreated(testPoolID)), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"labels":             map[string]string{"team": "platform"},
						"taints":             []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoSchedule"}},
						"security_group_ids": []string{"sg-1", "sg-2"},
					}), nil, nil).Once()

				return mc
			},
			NewConfig: poolConfig(map[string]interface{}{
				"labels":             map[string]interface{}{"team": "platform"},
				"taints":             []interface{}{taint("dedicated", "gpu", "NoSchedule")},
				"security_group_ids": []interface{}{"sg-1", "sg-2"},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)

				require.Equal(t, map[string]string{"team": "platform"}, sent.Labels)
				require.Equal(t, []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoSchedule"}}, sent.Taints)
				require.Equal(t, []string{"sg-1", "sg-2"}, sent.SecurityGroupIds)

				support.RequireStateAttrs(t, state, map[string]string{
					"labels.team":          "platform",
					"security_group_ids.0": "sg-1",
					"security_group_ids.1": "sg-2",
					"taints.#":             "1",
				})
			},
		},
		{
			Name: "an explicit node count of zero never reaches the request",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Run(capture).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(poolCreated(testPoolID)), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{"node_count": 3}), nil, nil).Once()

				return mc
			},
			NewConfig: poolConfig(map[string]interface{}{"node_count": 0}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.NotContains(t, marshal(t, sent), `"node_count"`,
					"d.GetOk cannot tell zero from absent and omitempty then drops the field off the wire")
				require.Equal(t, "3", state.Attributes["node_count"],
					"the server picks its own size and the read records it, so the plan never converges")
			},
		},
		{
			Name: "a create api error surfaces and leaves no state",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Return(nil, statusResponse(409), errors.New("pool with that name already exists")).Once()

				return mc
			},
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name: "a create task that ends in error orphans the pool",
			Op:   support.OpApply,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				require.Nil(t, state)
				mc.MKaaS.AssertNotCalled(t, "PoolGet", mock.Anything, mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSPoolResource_CreatePanics(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	t.Run("a create answer without task ids crashes the provider", func(t *testing.T) {
		t.Parallel()

		mc := mkaasmock.NewMockedMKaaS()
		mkaasmock.AllowProjectResolution(mc, testProjectID)
		mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
			Return(taskResponse(), nil, nil).Once()

		require.Panics(t, func() {
			support.ApplyConfig(t, t.Context(), resource, nil, poolConfig(), mc.TestMeta())
		})
	})

	t.Run("a finished create task that carries no pool id crashes the provider", func(t *testing.T) {
		t.Parallel()

		mc := mkaasmock.NewMockedMKaaS()
		mkaasmock.AllowProjectResolution(mc, testProjectID)
		mc.MKaaS.On("PoolCreate", mock.Anything, testClusterID, mock.Anything).
			Return(taskResponse(testTaskID), nil, nil).Once()
		mc.Tasks.On("Get", mock.Anything, testTaskID).
			Return(finishedTask(nil), nil, nil).Once()

		require.Panics(t, func() {
			support.ApplyConfig(t, t.Context(), resource, nil, poolConfig(), mc.TestMeta())
		})
	})
}

func TestIntegrationMKaaSPoolResource_Read(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "read maps every scalar, collection and computed attribute back",
			Op:        support.OpRead,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"name":               "renamed-out-of-band",
						"node_count":         5,
						"labels":             map[string]string{"team": "platform"},
						"taints":             []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoExecute"}},
						"security_group_ids": []string{"sg-1"},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"name":                 "renamed-out-of-band",
					"node_count":           "5",
					"current_node_count":   "5",
					"labels.team":          "platform",
					"security_group_ids.0": "sg-1",
					"taints.#":             "1",
					"state":                "ACTIVE",
					"status":               "Provisioned",
				})
			},
		},
		{
			Name:      "read of an autoscaled pool fills scale policy and refuses to touch the node count",
			Op:        support.OpRead,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"autoscaling_enabled": true,
						"min_node_count":      2,
						"max_node_count":      5,
						"node_count":          4,
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"scale_policy.0.auto_scale.0.min_node_count": "2",
					"scale_policy.0.auto_scale.0.max_node_count": "5",
					"current_node_count":                         "4",
					"node_count":                                 "2",
				})
			},
		},
		{
			Name:      "read keeps a deleted pool in state forever",
			Op:        support.OpRead,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(nil, notFound(), errors.New("pool not found")).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testPoolIDStr)
			},
		},
		{
			Name:         "read rejects a non numeric id before touching the api",
			Op:           support.OpRead,
			CurrentID:    "not-a-number",
			CurrentState: poolState(),
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				return mc
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, `invalid pool id "not-a-number"`)
				mc.MKaaS.AssertNotCalled(t, "PoolGet", mock.Anything, mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSPoolResource_Update(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	var sentLabels edgecloud.MKaaSPoolUpdateLabelsRequest

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "a flavor change is refused and still lands in state",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				return mkaasmock.NewMockedMKaaS()
			},
			CurrentState: poolState(),
			NewConfig:    poolConfig(map[string]interface{}{"flavor": "g1-standard-4-8"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags,
					"MKaaS pool update is not supported for these fields: [flavor]")
				require.Equal(t, "g1-standard-4-8", state.Attributes["flavor"],
					"the refused value is still written into state")
			},
		},
		{
			Name:      "a rename sends the name and waits for its task",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateName", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateNameRequest{Name: ptr("renamed")}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{"name": "renamed"}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			NewConfig:    poolConfig(map[string]interface{}{"name": "renamed"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"name": "renamed"})
			},
		},
		{
			Name:      "a node count change scales a pool that is not autoscaled",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateNodeCount", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateScaleRequest{NodeCount: ptr(5)}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{"node_count": 5}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			NewConfig:    poolConfig(map[string]interface{}{"node_count": 5}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"node_count":         "5",
					"current_node_count": "5",
				})
			},
		},
		{
			Name:      "a security group change replaces the whole list",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateSecurityGroups", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateSecurityGroupsRequest{SecurityGroupIds: []string{"sg-2", "sg-3"}}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"security_group_ids": []string{"sg-2", "sg-3"},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"security_group_ids": []interface{}{"sg-1"},
			}),
			NewConfig: poolConfig(map[string]interface{}{
				"security_group_ids": []interface{}{"sg-2", "sg-3"},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"security_group_ids.#": "2",
					"security_group_ids.0": "sg-2",
					"security_group_ids.1": "sg-3",
				})
			},
		},
		{
			Name:      "clearing the security group list sends an explicit empty array",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateSecurityGroups", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateSecurityGroupsRequest{SecurityGroupIds: []string{}}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(nil), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"security_group_ids": []interface{}{"sg-1"},
			}),
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "0", state.Attributes["security_group_ids.#"])
			},
		},
		{
			Name:      "a labels change replaces the whole map",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateLabels", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateLabelsRequest{Labels: map[string]string{"team": "infra"}}).
					Run(func(args mock.Arguments) {
						sentLabels = args.Get(3).(edgecloud.MKaaSPoolUpdateLabelsRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"labels": map[string]string{"team": "infra"},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"labels": map[string]interface{}{"team": "platform"},
			}),
			NewConfig: poolConfig(map[string]interface{}{
				"labels": map[string]interface{}{"team": "infra"},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, `{"labels":{"team":"infra"}}`, marshal(t, sentLabels))
				support.RequireStateAttrs(t, state, map[string]string{"labels.team": "infra"})
			},
		},
		{
			Name:      "dropping every label sends a body with no labels key at all",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateLabels", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateLabelsRequest{Labels: map[string]string{}}).
					Run(func(args mock.Arguments) {
						sentLabels = args.Get(3).(edgecloud.MKaaSPoolUpdateLabelsRequest)
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"labels": map[string]string{"team": "platform"},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"labels": map[string]interface{}{"team": "platform"},
			}),
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "{}", marshal(t, sentLabels),
					"omitempty drops the empty map, so the api is asked to change nothing")
				require.Equal(t, "platform", state.Attributes["labels.team"],
					"the labels survive on the api side and the plan never converges")
			},
		},
		{
			Name:      "a taints change replaces the whole set",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateTaints", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateTaintsRequest{
						Taints: []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoExecute"}},
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"taints": []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoExecute"}},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"taints": []interface{}{taint("dedicated", "gpu", "NoSchedule")},
			}),
			NewConfig: poolConfig(map[string]interface{}{
				"taints": []interface{}{taint("dedicated", "gpu", "NoExecute")},
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, "1", state.Attributes["taints.#"])
			},
		},
		{
			Name:      "enabling autoscaling patches autoscaling only",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateAutoscaling", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateAutoscalingRequest{
						EnableAutoscaling: ptr(true),
						MinNodeCount:      ptr(2),
						MaxNodeCount:      ptr(5),
					}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"autoscaling_enabled": true,
						"min_node_count":      2,
						"max_node_count":      5,
						"node_count":          4,
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			NewConfig: merge(poolConfig(), map[string]interface{}{
				"node_count":   nil,
				"scale_policy": scalePolicy(2, 5),
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				mc.MKaaS.AssertNotCalled(t, "PoolUpdateNodeCount",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				require.Equal(t, "2", state.Attributes["node_count"],
					"node_count freezes at its old value while the autoscaler owns the size")
			},
		},
		{
			Name:      "disabling autoscaling leaves the pool at the size the autoscaler chose",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateAutoscaling", mock.Anything, testClusterID, testPoolID,
					edgecloud.MKaaSPoolUpdateAutoscalingRequest{EnableAutoscaling: ptr(false)}).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()
				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{"node_count": 7}), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(map[string]interface{}{
				"scale_policy": scalePolicy(2, 9),
			}),
			NewConfig: poolConfig(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				mc.MKaaS.AssertNotCalled(t, "PoolUpdateNodeCount",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				require.Equal(t, "7", state.Attributes["node_count"],
					"the apply reports success with seven live nodes, only a second apply scales down")
			},
		},
		{
			Name:      "an update whose task fails surfaces the wait error after the earlier calls landed",
			Op:        support.OpApply,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolUpdateName", mock.Anything, testClusterID, testPoolID, mock.Anything).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			NewConfig:    poolConfig(map[string]interface{}{"name": "renamed"}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				mc.MKaaS.AssertNotCalled(t, "PoolGet", mock.Anything, mock.Anything, mock.Anything)
				require.Equal(t, "renamed", state.Attributes["name"],
					"the name the api never confirmed is recorded anyway")
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSPoolResource_Delete(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "delete waits for the task and clears the id",
			Op:        support.OpDelete,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolDelete", mock.Anything, testClusterID, testPoolID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(finishedTask(nil), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Nil(t, state)
			},
		},
		{
			Name:      "deleting an already deleted pool fails and keeps it in state",
			Op:        support.OpDelete,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolDelete", mock.Anything, testClusterID, testPoolID).
					Return(nil, notFound(), errors.New("pool not found")).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireStateID(t, state, testPoolIDStr)
			},
		},
		{
			Name:      "a delete task in the error state reports the sdk wording, never the resource wording",
			Op:        support.OpDelete,
			CurrentID: testPoolIDStr,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolDelete", mock.Anything, testClusterID, testPoolID).
					Return(taskResponse(testTaskID), nil, nil).Once()
				mc.Tasks.On("Get", mock.Anything, testTaskID).
					Return(failedTask(), nil, nil).Once()

				return mc
			},
			CurrentState: poolState(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "task with error state")
				for _, d := range diags {
					require.NotContains(t, d.Summary, "cannot delete MKaaS Pool with ID",
						"the TaskStateError branch below the wait is unreachable")
				}
			},
		},
		{
			Name:         "delete rejects a non numeric id",
			Op:           support.OpDelete,
			CurrentID:    "not-a-number",
			CurrentState: poolState(),
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				return mc
			},
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, mc *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, `invalid pool id "not-a-number"`)
				mc.MKaaS.AssertNotCalled(t, "PoolDelete", mock.Anything, mock.Anything, mock.Anything)
			},
		},
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}

func TestIntegrationMKaaSPoolResource_DeletePanicsWithoutATaskID(t *testing.T) {
	t.Parallel()

	resource := mkaasResource(t, mkaassvc.MKaaSPoolResource)

	mc := mkaasmock.NewMockedMKaaS()
	mkaasmock.AllowProjectResolution(mc, testProjectID)
	mc.MKaaS.On("PoolDelete", mock.Anything, testClusterID, testPoolID).
		Return(taskResponse(), nil, nil).Once()

	state := support.NewState(t, resource, poolState(), testPoolIDStr)
	data := support.NewResourceDataFromState(t, resource, state)

	require.Panics(t, func() {
		resource.DeleteContext(t.Context(), data, mc.TestMeta())
	})
}
