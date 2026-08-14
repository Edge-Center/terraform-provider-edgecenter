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

func poolLookup(parts ...map[string]interface{}) map[string]interface{} {
	base := merge(projectRegion(), map[string]interface{}{
		"cluster_id": testClusterID,
		"pool_id":    testPoolID,
	})

	return merge(base, parts...)
}

func TestIntegrationMKaaSPoolDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := mkaasDataSource(t, mkaassvc.MKaaSPoolDataSource)

	cases := []support.ResourceCase[*mkaasmock.MockedMKaaS]{
		{
			Name:      "read fills every attribute from the pool payload",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{
						"node_count":         5,
						"labels":             map[string]string{"team": "platform"},
						"taints":             []edgecloud.MKaaSTaint{{Key: "dedicated", Value: "gpu", Effect: "NoSchedule"}},
						"security_group_ids": []string{"sg-1"},
					}), nil, nil).Once()

				return mc
			},
			CurrentState: poolLookup(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testPoolIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"cluster_id":           testClusterIDStr,
					"pool_id":              testPoolIDStr,
					"name":                 testPoolName,
					"flavor":               testFlavor,
					"node_count":           "5",
					"current_node_count":   "5",
					"volume_size":          "20",
					"volume_type":          string(edgecloud.VolumeTypeStandard),
					"labels.team":          "platform",
					"security_group_ids.0": "sg-1",
					"taints.#":             "1",
					"state":                "ACTIVE",
					"status":               "Provisioned",
				})
			},
		},
		{
			Name:      "read flattens autoscaling into the scale policy block",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
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
			CurrentState: poolLookup(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{
					"scale_policy.0.auto_scale.0.min_node_count": "2",
					"scale_policy.0.auto_scale.0.max_node_count": "5",
					"node_count":         "4",
					"current_node_count": "4",
				})
			},
		},
		{
			Name:      "a get failure is wrapped with both ids",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(nil, notFound(), errors.New("pool not found")).Once()

				return mc
			},
			CurrentState: poolLookup(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags,
					"failed to get MKaaS pool 7 in cluster 42")
			},
		},
		{
			Name:      "zero ids are passed straight to the api",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, 0, 0).
					Return(nil, statusResponse(400), errors.New("bad request")).Once()

				return mc
			},
			CurrentState: merge(projectRegion(), map[string]interface{}{
				"cluster_id": 0,
				"pool_id":    0,
			}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireErrorDiagContains(t, diags, "failed to get MKaaS pool 0 in cluster 0")
			},
		},
		{
			Name:      "the data source reports the same live count under two names",
			Op:        support.OpRead,
			CurrentID: unsetDataSourceID,
			Prepare: func() *mkaasmock.MockedMKaaS {
				mc := mkaasmock.NewMockedMKaaS()

				mkaasmock.AllowProjectResolution(mc, testProjectID)

				mc.MKaaS.On("PoolGet", mock.Anything, testClusterID, testPoolID).
					Return(samplePool(map[string]interface{}{"node_count": 9}), nil, nil).Once()

				return mc
			},
			CurrentState: poolLookup(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *mkaasmock.MockedMKaaS) {
				support.RequireNoErrorDiags(t, diags)
				require.Equal(t, state.Attributes["node_count"], state.Attributes["current_node_count"])
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*mkaasmock.MockedMKaaS])
}
