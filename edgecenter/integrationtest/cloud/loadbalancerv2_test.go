//go:build integration

package edgecenter_test

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud"
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const testLoadBalancerID = "test-lb-id"

func sampleLoadBalancer(id string) *edgecloud.Loadbalancer {
	return &edgecloud.Loadbalancer{
		ID:           id,
		Name:         "test-lb",
		ProjectID:    testProjectID,
		RegionID:     testRegionID,
		Region:       "test-region",
		VipAddress:   net.ParseIP("10.0.0.1"),
		Flavor:       edgecloud.Flavor{FlavorName: "lb1-1-2"},
		ProvisioningStatus: edgecloud.ProvisioningStatusActive,
		OperatingStatus:    edgecloud.OperatingStatusOnline,
	}
}

func sampleLoadBalancerUpdated(id string) *edgecloud.Loadbalancer {
	lb := sampleLoadBalancer(id)
	lb.Name = "new-lb-name"
	return lb
}

func loadbalancerCreateCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.LoadbalancerCreateRequest) bool {
			return req.Name == "test-lb"
		}),
	).Return(&edgecloud.TaskResponse{Tasks: []string{"task-lb-1"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-lb-1").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
			CreatedResources: map[string]interface{}{
				"loadbalancers": []interface{}{lbID},
			},
		}, nil, nil)

	mc.Loadbalancers.On("Get", mock.Anything, lbID).
		Return(sampleLoadBalancer(lbID), nil, nil)

	mc.Loadbalancers.On("MetadataList", mock.Anything, lbID).
		Return([]edgecloud.MetadataDetailed{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name": "test-lb",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, lbID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "test-lb",
				"flavor":      "lb1-1-2",
				"vip_address": "10.0.0.1",
			})
		},
	}
}

func loadbalancerReadCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Get", mock.Anything, lbID).
		Return(sampleLoadBalancer(lbID), nil, nil)

	mc.Loadbalancers.On("MetadataList", mock.Anything, lbID).
		Return([]edgecloud.MetadataDetailed{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing loadbalancer",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, lbID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "test-lb",
				"flavor":      "lb1-1-2",
				"vip_address": "10.0.0.1",
			})
		},
	}
}

func loadbalancerReadNotFoundCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Get", mock.Anything, lbID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
		},
	}
}

func loadbalancerUpdateNameCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("Rename", mock.Anything, lbID,
		mock.MatchedBy(func(req *edgecloud.Name) bool {
			return req.Name == "new-lb-name"
		}),
	).Return(sampleLoadBalancerUpdated(lbID), nil, nil)

	mc.Loadbalancers.On("Get", mock.Anything, lbID).
		Return(sampleLoadBalancerUpdated(lbID), nil, nil)

	mc.Loadbalancers.On("MetadataList", mock.Anything, lbID).
		Return([]edgecloud.MetadataDetailed{}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("new-lb-name"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, lbID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "new-lb-name",
			})
		},
	}
}

func loadbalancerUpdateMetadataCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Loadbalancers.On("MetadataUpdate", mock.Anything, lbID,
		mock.MatchedBy(func(meta *edgecloud.Metadata) bool {
			return len(*meta) == 1 && (*meta)["env"] == "prod"
		}),
	).Return(nil, nil)

	lbWithMeta := sampleLoadBalancer(lbID)
	lbWithMeta.MetadataDetailed = []edgecloud.MetadataDetailed{
		{Key: "env", Value: "prod", ReadOnly: false},
	}

	mc.Loadbalancers.On("Get", mock.Anything, lbID).
		Return(lbWithMeta, nil, nil)

	mc.Loadbalancers.On("MetadataList", mock.Anything, lbID).
		Return([]edgecloud.MetadataDetailed{
			{Key: "env", Value: "prod", ReadOnly: false},
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update metadata",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
			cloud.WithMetadata(map[string]string{"env": "prod"}),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, lbID)
		},
	}
}

func loadbalancerDeleteCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Delete", mock.Anything, lbID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-lb-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-lb-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete loadbalancer",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func loadbalancerCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.LoadbalancerCreateRequest) bool {
			return req.Name == "fail-lb"
		}),
	).Return(nil, nil, fmt.Errorf("api error: create failed"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"name": "fail-lb",
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create failed")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func loadbalancerDeleteTaskErrorCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Delete", mock.Anything, lbID).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-lb-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-lb-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
			Error: strPtr("internal error"),
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete task error",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, lbID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func loadbalancerDeleteNotFoundCase(lbID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Loadbalancers.On("Delete", mock.Anything, lbID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete non-existent (404)",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: lbID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-lb"),
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			require.Nil(t, state, "state must be nil after deleting non-existent resource")
		},
	}
}

func TestIntegrationLoadBalancerV2_TableDriven(t *testing.T) {
	t.Parallel()

	resource := provider.Provider().ResourcesMap["edgecenter_loadbalancerv2"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		loadbalancerCreateCase(testLoadBalancerID),
		loadbalancerReadCase(testLoadBalancerID),
		loadbalancerReadNotFoundCase(testLoadBalancerID),
		loadbalancerUpdateNameCase(testLoadBalancerID),
		loadbalancerUpdateMetadataCase(testLoadBalancerID),
		loadbalancerDeleteCase(testLoadBalancerID),
		loadbalancerCreateAPIFailureCase(),
		loadbalancerDeleteTaskErrorCase(testLoadBalancerID),
		loadbalancerDeleteNotFoundCase(testLoadBalancerID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
