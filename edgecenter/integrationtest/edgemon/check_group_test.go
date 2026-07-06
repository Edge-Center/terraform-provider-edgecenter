//go:build integration

package edgemon_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenteredgemon-go/checkgroup"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckGroupID = 106

func checkGroupConfig(name string) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
	}
}

func checkGroupCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checkgroup.Request) bool {
			return req.Name == "tf-group"
		}),
	).Return(&checkgroup.Response{ID: testCheckGroupID, Name: "tf-group"}, nil)

	mc.CheckGroup.On("Get", mock.Anything, testCheckGroupID).
		Return(&checkgroup.Response{ID: testCheckGroupID, Name: "tf-group"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckGroupID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "tf-group",
			})
		},
	}
}

func checkGroupReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Get", mock.Anything, testCheckGroupID).
		Return(&checkgroup.Response{ID: testCheckGroupID, Name: "tf-group"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check group",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckGroupID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "tf-group",
			})
		},
	}
}

func checkGroupUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Update", mock.Anything, testCheckGroupID,
		mock.MatchedBy(func(req *checkgroup.Request) bool {
			return req.Name == "tf-group-2"
		}),
	).Return(&checkgroup.Response{ID: testCheckGroupID, Name: "tf-group-2"}, nil)

	mc.CheckGroup.On("Get", mock.Anything, testCheckGroupID).
		Return(&checkgroup.Response{ID: testCheckGroupID, Name: "tf-group-2"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update name",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		NewConfig:    checkGroupConfig("tf-group-2"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "tf-group-2",
			})
		},
	}
}

func checkGroupCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: invalid name"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid name")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkGroupDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Delete", mock.Anything, testCheckGroupID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check group",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkGroupDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Delete", mock.Anything, testCheckGroupID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckGroupID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkGroupDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Delete", mock.Anything, testCheckGroupID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkGroupReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckGroup.On("Get", mock.Anything, testCheckGroupID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckGroupID),
		CurrentState: checkGroupConfig("tf-group"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func TestIntegrationCheckGroup_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_group")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkGroupCreateCase(),
		checkGroupReadCase(),
		checkGroupUpdateCase(),
		checkGroupCreateAPIFailureCase(),
		checkGroupDeleteCase(),
		checkGroupDeleteAPIFailureCase(),
		checkGroupDeleteNotFoundCase(),
		checkGroupReadNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
