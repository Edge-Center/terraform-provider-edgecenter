//go:build unit

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
	cloudmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cloud/mock"
)

const testProjectResourceID = 100

func sampleProject(id int, name string) *edgecloud.Project {
	return &edgecloud.Project{
		ID:          id,
		Name:        name,
		Description: "test project",
		ClientID:    1,
		State:       edgecloud.ProjectStateActive,
		CreatedAt:   "2024-01-01T00:00:00Z",
		IsDefault:   false,
	}
}

func projectCreateCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ProjectCreateRequest) bool {
			return req.Name == "test-project" && req.Description == "test project"
		}),
	).Return(sampleProject(projID, "test-project"), nil, nil)

	mc.Projects.On("Get", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(sampleProject(projID, "test-project"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "successful create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: map[string]interface{}{
			"name":        "test-project",
			"description": "test project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", projID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "test-project",
				"description": "test project",
				"state":       string(edgecloud.ProjectStateActive),
				"is_default":  "false",
			})
		},
	}
}

func projectReadCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Get", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(sampleProject(projID, "test-project"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing project",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fmt.Sprintf("%d", projID),
		CurrentState: map[string]interface{}{
			"name": "test-project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", projID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":        "test-project",
				"description": "test project",
				"client_id":   "1",
				"state":       string(edgecloud.ProjectStateActive),
				"is_default":  "false",
			})
		},
	}
}

func projectUpdateNameCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Update", mock.Anything, fmt.Sprintf("%d", projID),
		mock.MatchedBy(func(req *edgecloud.ProjectUpdateRequest) bool {
			return req.Name == "updated-project" && req.Description == "test project"
		}),
	).Return((*edgecloud.Project)(nil), nil, nil)

	updated := sampleProject(projID, "updated-project")

	mc.Projects.On("Get", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update project name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fmt.Sprintf("%d", projID),
		CurrentState: map[string]interface{}{
			"name":        "test-project",
			"description": "test project",
		},
		NewConfig: map[string]interface{}{
			"name":        "updated-project",
			"description": "test project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", projID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "updated-project",
			})
		},
	}
}

func projectDeleteCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Delete", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-proj-del"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-proj-del").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateFinished,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete project",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fmt.Sprintf("%d", projID),
		CurrentState: map[string]interface{}{
			"name": "test-project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func projectCreateAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.ProjectCreateRequest) bool {
			return req.Name == "fail-project"
		}),
	).Return(nil, nil, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on create",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: map[string]interface{}{
			"name": "fail-project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func projectReadNonExistentCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Get", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fmt.Sprintf("%d", projID),
		CurrentState: map[string]interface{}{
			"name": "test-project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil when resource not found")
		},
	}
}

func projectDeleteTaskErrorCase(projID int) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)

	mc.Projects.On("Delete", mock.Anything, fmt.Sprintf("%d", projID)).
		Return(&edgecloud.TaskResponse{Tasks: []string{"task-proj-err"}}, nil, nil)

	mc.Tasks.On("Get", mock.Anything, "task-proj-err").
		Return(&edgecloud.Task{
			State: edgecloud.TaskStateError,
		}, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "task error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: fmt.Sprintf("%d", projID),
		CurrentState: map[string]interface{}{
			"name": "test-project",
		},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", projID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func TestUnitProject_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_project"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		projectCreateCase(testProjectResourceID),
		projectReadCase(testProjectResourceID),
		projectUpdateNameCase(testProjectResourceID),
		projectDeleteCase(testProjectResourceID),
		projectCreateAPIFailureCase(),
		projectReadNonExistentCase(testProjectResourceID),
		projectDeleteTaskErrorCase(testProjectResourceID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
