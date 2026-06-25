//go:build integration

package edgecenter_test

import (
	"fmt"
	"net/http"
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

const testServerGroupID = "sg-id"

func sampleServerGroup(id, name string, policy edgecloud.ServerGroupPolicy) *edgecloud.ServerGroup {
	return &edgecloud.ServerGroup{
		ID:        id,
		Name:      name,
		Policy:    policy,
		Instances: []edgecloud.ServerGroupInstance{},
		ProjectID: testProjectID,
		RegionID:  testRegionID,
	}
}

func serverGroupCreateCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.ServerGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ServerGroupCreateRequest) bool {
			return req.Name == "test-sg" && req.Policy == edgecloud.ServerGroupPolicyAntiAffinity
		}),
	).Return(sampleServerGroup(sgID, "test-sg", edgecloud.ServerGroupPolicyAntiAffinity), nil, nil)

	mc.ServerGroups.On("Get", mock.Anything, sgID).
		Return(sampleServerGroup(sgID, "test-sg", edgecloud.ServerGroupPolicyAntiAffinity), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{"policy": "anti-affinity"},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, sgID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name":   "test-sg",
				"policy": "anti-affinity",
			})
		},
	}
}

func serverGroupDeleteCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ServerGroups.On("Delete", mock.Anything, sgID).
		Return(nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete server group",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{"policy": "anti-affinity"},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func serverGroupCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ServerGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ServerGroupCreateRequest) bool {
			return req.Name == "fail-sg"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("fail-sg"),
			map[string]interface{}{"policy": "anti-affinity"},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func serverGroupReadNonExistentCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ServerGroups.On("Get", mock.Anything, sgID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{"policy": "anti-affinity"},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
			require.NotNil(t, state, "state must not be cleared when read fails")
			require.Equal(t, sgID, state.ID)
		},
	}
}

func serverGroupDeleteAPIFailureCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.ServerGroups.On("Delete", mock.Anything, sgID).
		Return(nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "API error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{"policy": "anti-affinity"},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, sgID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestIntegrationServerGroup_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_servergroup"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		serverGroupCreateCase(testServerGroupID),
		serverGroupDeleteCase(testServerGroupID),
		serverGroupCreateAPIFailureCase(),
		serverGroupReadNonExistentCase(testServerGroupID),
		serverGroupDeleteAPIFailureCase(testServerGroupID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
