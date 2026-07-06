//go:build integration

package edgemon_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenteredgemon-go/checks"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkrabbitmq"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemon "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const testCheckRabbitMQID = 105

func baseCheckRabbitMQConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":          "tf-rabbitmq",
		"enabled":       true,
		"place":         "country",
		"entities":      []interface{}{1, 2},
		"ip":            "1.1.1.1",
		"port":          5672,
		"username":      "user",
		"password":      "pass",
		"vhost":         "/",
		"interval":      120,
		"check_timeout": 2,
		"retries":       3,
	}
}

func sampleCheckRabbitMQ(name, place, vhost string, entities []int) *checkrabbitmq.Response {
	return &checkrabbitmq.Response{
		Name:         name,
		Enabled:      1,
		Place:        place,
		Entities:     entities,
		IP:           "1.1.1.1",
		Port:         5672,
		Username:     "user",
		Password:     "pass",
		Vhost:        vhost,
		Interval:     120,
		CheckTimeout: 2,
		Retries:      3,
	}
}

func checkRabbitMQCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Create", mock.Anything,
		mock.MatchedBy(func(req *checkrabbitmq.Request) bool {
			return req.Name == "tf-rabbitmq" && req.IP == "1.1.1.1" &&
				req.Port == 5672 && req.Username == "user" && req.Vhost == "/" &&
				req.Enabled == 1 && req.Place == "country"
		}),
	).Return(&checks.CreateResponse{ID: testCheckRabbitMQID}, nil)

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(sampleCheckRabbitMQ("tf-rabbitmq", "country", "/", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckRabbitMQID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-rabbitmq",
				"enabled":    "true",
				"place":      "country",
				"ip":         "1.1.1.1",
				"port":       "5672",
				"username":   "user",
				"vhost":      "/",
				"entities.#": "2",
				"entities.0": "1",
				"entities.1": "2",
			})
		},
	}
}

func checkRabbitMQCreatePlaceAllCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Create", mock.Anything, mock.Anything).
		Return(&checks.CreateResponse{ID: testCheckRabbitMQID}, nil)

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(sampleCheckRabbitMQ("tf-rabbitmq", "all", "/", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:    "place all clears entities on read",
		Op:      support.OpApply,
		Prepare: func() *edgemonmock.MockedRMON { return mc },
		NewConfig: edgemon.Merge(baseCheckRabbitMQConfig(), map[string]interface{}{
			"place":    "all",
			"entities": []interface{}{},
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"place":      "all",
				"entities.#": "0",
			})
		},
	}
}

func checkRabbitMQReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(sampleCheckRabbitMQ("tf-rabbitmq", "country", "/", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing check",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testCheckRabbitMQID))
			support.RequireStateAttrs(t, state, map[string]string{
				"name":       "tf-rabbitmq",
				"vhost":      "/",
				"entities.#": "2",
			})
		},
	}
}

func checkRabbitMQUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Update", mock.Anything, testCheckRabbitMQID,
		mock.MatchedBy(func(req *checkrabbitmq.Request) bool {
			return req.Vhost == "/app"
		}),
	).Return(nil)

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(sampleCheckRabbitMQ("tf-rabbitmq", "country", "/app", []int{1, 2}), nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update vhost",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		NewConfig: edgemon.Merge(baseCheckRabbitMQConfig(), map[string]interface{}{
			"vhost": "/app",
		}),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"vhost": "/app",
			})
		},
	}
}

func checkRabbitMQCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Create", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func checkRabbitMQReadErrorCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(nil, fmt.Errorf("api error: internal server error"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read surfaces non-404 error",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "internal server error")
		},
	}
}

func checkRabbitMQDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Delete", mock.Anything, testCheckRabbitMQID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete check",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func checkRabbitMQDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Delete", mock.Anything, testCheckRabbitMQID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testCheckRabbitMQID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func checkRabbitMQReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Get", mock.Anything, testCheckRabbitMQID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func checkRabbitMQDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.CheckRabbitMQ.On("Delete", mock.Anything, testCheckRabbitMQID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testCheckRabbitMQID),
		CurrentState: baseCheckRabbitMQConfig(),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func TestIntegrationCheckRabbitMQ_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_check_rabbitmq")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		checkRabbitMQCreateCase(),
		checkRabbitMQCreatePlaceAllCase(),
		checkRabbitMQReadCase(),
		checkRabbitMQUpdateCase(),
		checkRabbitMQCreateAPIFailureCase(),
		checkRabbitMQReadErrorCase(),
		checkRabbitMQDeleteCase(),
		checkRabbitMQDeleteAPIFailureCase(),
		checkRabbitMQReadNotFoundCase(),
		checkRabbitMQDeleteNotFoundCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
