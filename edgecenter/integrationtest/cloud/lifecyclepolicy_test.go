//go:build integration

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const testLCPID = 1

func sampleLifecyclePolicy(id int, name string) *edgecloud.LifeCyclePolicy {
	return &edgecloud.LifeCyclePolicy{
		ID:        id,
		Name:      name,
		Status:    edgecloud.LifeCyclePolicyStatusActive,
		Action:    edgecloud.LifeCyclePolicyActionVolumeSnapshot,
		UserID:    1,
		ProjectID: testProjectID,
		RegionID:  testRegionID,
		Volumes: []edgecloud.LifeCyclePolicyVolume{
			{ID: "vol-1", Name: "test-vol"},
		},
		Schedules: []edgecloud.LifeCyclePolicySchedule{
			edgecloud.LifeCyclePolicyIntervalSchedule{
				LifeCyclePolicyCommonSchedule: edgecloud.LifeCyclePolicyCommonSchedule{
					Type:                 edgecloud.LifeCyclePolicyScheduleTypeInterval,
					ID:                   "sched-1",
					MaxQuantity:          5,
					ResourceNameTemplate: "snap-{volume_id}",
					RetentionTime:        nil,
				},
				Hours: 1,
			},
		},
	}
}

func lifecyclePolicyCreateCase(lcpID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.LifeCyclePolicies.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyCreateRequest) bool {
			return req.Name == "test-lcp"
		}),
	).Return(sampleLifecyclePolicy(lcpID, "test-lcp"), nil, nil)

	mc.LifeCyclePolicies.On("Get", mock.Anything, lcpID,
		mock.MatchedBy(func(opts *edgecloud.LifeCyclePolicyGetOptions) bool {
			return opts.NeedVolumes
		}),
	).Return(sampleLifecyclePolicy(lcpID, "test-lcp"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"volume": []interface{}{
					map[string]interface{}{
						"id": "vol-1",
					},
				},
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "1")
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   "test-lcp",
				"status": "active",
				"action": "volume_snapshot",
			})
		},
	}
}

func lifecyclePolicyUpdateNameCase(lcpID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.LifeCyclePolicies.On("Update", mock.Anything, lcpID,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyUpdateRequest) bool {
			return req.Name == "updated-lcp"
		}),
	).Return((*edgecloud.LifeCyclePolicy)(nil), nil, nil)

	updated := sampleLifecyclePolicy(lcpID, "updated-lcp")
	updated.Schedules = []edgecloud.LifeCyclePolicySchedule{
		edgecloud.LifeCyclePolicyIntervalSchedule{
			LifeCyclePolicyCommonSchedule: edgecloud.LifeCyclePolicyCommonSchedule{
				Type:                 edgecloud.LifeCyclePolicyScheduleTypeInterval,
				ID:                   "sched-1",
				MaxQuantity:          10,
				ResourceNameTemplate: "snap-{volume_id}",
				RetentionTime:        nil,
			},
			Hours: 2,
		},
	}

	mc.LifeCyclePolicies.On("RemoveSchedules", mock.Anything, lcpID,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyRemoveSchedulesRequest) bool {
			return len(req.ScheduleIDs) == 1 && req.ScheduleIDs[0] == "sched-1"
		}),
	).Return((*edgecloud.LifeCyclePolicy)(nil), nil, nil)

	mc.LifeCyclePolicies.On("AddSchedules", mock.Anything, lcpID,
		mock.Anything,
	).Return((*edgecloud.LifeCyclePolicy)(nil), nil, nil)

	mc.LifeCyclePolicies.On("Get", mock.Anything, lcpID,
		mock.MatchedBy(func(opts *edgecloud.LifeCyclePolicyGetOptions) bool {
			return opts.NeedVolumes
		}),
	).Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update name + schedules",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: "1",
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"volume": []interface{}{
					map[string]interface{}{
						"id": "vol-1",
					},
				},
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"id":           "sched-1",
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("updated-lcp"),
			map[string]interface{}{
				"volume": []interface{}{
					map[string]interface{}{
						"id": "vol-1",
					},
				},
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 10,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 2,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "1")
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "updated-lcp",
			})
		},
	}
}

func lifecyclePolicyUpdateVolumesCase(lcpID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.LifeCyclePolicies.On("RemoveVolumes", mock.Anything, lcpID,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyRemoveVolumesRequest) bool {
			return len(req.VolumeIds) == 1 && req.VolumeIds[0] == "vol-1"
		}),
	).Return((*edgecloud.LifeCyclePolicy)(nil), nil, nil)

	mc.LifeCyclePolicies.On("AddVolumes", mock.Anything, lcpID,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyAddVolumesRequest) bool {
			return len(req.VolumeIds) == 1 && req.VolumeIds[0] == "vol-2"
		}),
	).Return((*edgecloud.LifeCyclePolicy)(nil), nil, nil)

	updated := sampleLifecyclePolicy(lcpID, "test-lcp")
	updated.Volumes = []edgecloud.LifeCyclePolicyVolume{
		{ID: "vol-2", Name: "test-vol-2"},
	}

	mc.LifeCyclePolicies.On("Get", mock.Anything, lcpID,
		mock.MatchedBy(func(opts *edgecloud.LifeCyclePolicyGetOptions) bool {
			return opts.NeedVolumes
		}),
	).Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update volumes (add/remove)",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: "1",
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"volume": []interface{}{
					map[string]interface{}{
						"id": "vol-1",
					},
				},
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"volume": []interface{}{
					map[string]interface{}{
						"id": "vol-2",
					},
				},
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, "1")
		},
	}
}

func lifecyclePolicyDeleteCase(lcpID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.LifeCyclePolicies.On("Delete", mock.Anything, lcpID).
		Return(nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete lifecycle policy",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: "1",
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func lifecyclePolicyCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.LifeCyclePolicies.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.LifeCyclePolicyCreateRequest) bool {
			return req.Name == "fail-lcp"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-lcp"),
			map[string]interface{}{
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func lifecyclePolicyValidationEmptyScheduleCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "validation: empty schedule",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "at least one 'schedule' should be set")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func lifecyclePolicyDeleteAPIFailureCase(lcpID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.LifeCyclePolicies.On("Delete", mock.Anything, lcpID).
		Return(nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "API error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: "1",
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lcp"),
			map[string]interface{}{
				"schedule": []interface{}{
					map[string]interface{}{
						"max_quantity": 5,
						"interval": []interface{}{
							map[string]interface{}{
								"hours": 1,
							},
						},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, "1", state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationLifecyclePolicy_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_lifecyclepolicy"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		lifecyclePolicyCreateCase(testLCPID),
		lifecyclePolicyUpdateNameCase(testLCPID),
		lifecyclePolicyUpdateVolumesCase(testLCPID),
		lifecyclePolicyDeleteCase(testLCPID),
		lifecyclePolicyCreateAPIFailureCase(),
		lifecyclePolicyValidationEmptyScheduleCase(),
		lifecyclePolicyDeleteAPIFailureCase(testLCPID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
