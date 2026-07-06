//go:build integration

package edgemon_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenteredgemon-go/channel"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	edgemonmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/edgemon/mock"
)

const (
	testChannelID = 55
	testReceiver  = "telegram"
)

func channelConfig(name, token string) map[string]interface{} {
	return map[string]interface{}{
		"receiver":     testReceiver,
		"channel_name": name,
		"token":        token,
	}
}

func channelCreateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Create", mock.Anything, testReceiver,
		mock.MatchedBy(func(req *channel.Request) bool {
			return req.Channel == "tf-channel" && req.Token == "tok-1"
		}),
	).Return(&channel.Response{ID: testChannelID, Channel: "tf-channel", Token: "tok-1"}, nil)

	mc.Channel.On("Get", mock.Anything, testReceiver, testChannelID).
		Return(&channel.Response{ID: testChannelID, Channel: "tf-channel", Token: "tok-1"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "successful create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testChannelID))
			support.RequireStateAttrs(t, state, map[string]string{
				"receiver":     testReceiver,
				"channel_name": "tf-channel",
				"token":        "tok-1",
			})
		},
	}
}

func channelReadCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Get", mock.Anything, testReceiver, testChannelID).
		Return(&channel.Response{ID: testChannelID, Channel: "tf-channel", Token: "tok-1"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read existing channel",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testChannelID))
			support.RequireStateAttrs(t, state, map[string]string{
				"channel_name": "tf-channel",
				"token":        "tok-1",
			})
		},
	}
}

func channelUpdateCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Update", mock.Anything, testReceiver, testChannelID,
		mock.MatchedBy(func(req *channel.Request) bool {
			return req.Token == "tok-2"
		}),
	).Return(nil)

	mc.Channel.On("Get", mock.Anything, testReceiver, testChannelID).
		Return(&channel.Response{ID: testChannelID, Channel: "tf-channel", Token: "tok-2"}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "update token",
		Op:           support.OpApply,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		NewConfig:    channelConfig("tf-channel", "tok-2"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"token": "tok-2",
			})
		},
	}
}

func channelCreateAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Create", mock.Anything, testReceiver, mock.Anything).
		Return(nil, fmt.Errorf("api error: invalid token"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:      "API error on create",
		Op:        support.OpApply,
		Prepare:   func() *edgemonmock.MockedRMON { return mc },
		NewConfig: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "invalid token")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func channelDeleteCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Delete", mock.Anything, testReceiver, testChannelID).Return(nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete channel",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func channelDeleteAPIFailureCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Delete", mock.Anything, testReceiver, testChannelID).
		Return(fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "API error on delete",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "server unavailable")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, fmt.Sprintf("%d", testChannelID), state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func channelDeleteNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Delete", mock.Anything, testReceiver, testChannelID).
		Return(fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "delete tolerates 404",
		Op:           support.OpDelete,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after delete")
		},
	}
}

func channelReadNotFoundCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Get", mock.Anything, testReceiver, testChannelID).
		Return(nil, fmt.Errorf("404 not found"))

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read clears state on 404",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "tok-1"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after read clears a missing resource")
		},
	}
}

func channelReadEmptyTokenCase() support.ResourceCase[*edgemonmock.MockedRMON] {
	mc := edgemonmock.NewMockedRMON()

	mc.Channel.On("Get", mock.Anything, testReceiver, testChannelID).
		Return(&channel.Response{ID: testChannelID, Channel: "tf-channel", Token: ""}, nil)

	return support.ResourceCase[*edgemonmock.MockedRMON]{
		Name:         "read keeps config token when API returns empty",
		Op:           support.OpRead,
		Prepare:      func() *edgemonmock.MockedRMON { return mc },
		CurrentID:    fmt.Sprintf("%d", testChannelID),
		CurrentState: channelConfig("tf-channel", "cfg-token"),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *edgemonmock.MockedRMON) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				"token": "cfg-token",
			})
		},
	}
}

func TestIntegrationChannel_TableDriven(t *testing.T) {
	t.Parallel()

	resource := rmonResource(t, "edgecenter_rmon_channel")

	cases := []support.ResourceCase[*edgemonmock.MockedRMON]{
		channelCreateCase(),
		channelReadCase(),
		channelUpdateCase(),
		channelCreateAPIFailureCase(),
		channelDeleteCase(),
		channelDeleteAPIFailureCase(),
		channelDeleteNotFoundCase(),
		channelReadNotFoundCase(),
		channelReadEmptyTokenCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*edgemonmock.MockedRMON])
}
