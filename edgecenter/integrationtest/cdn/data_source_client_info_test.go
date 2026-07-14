//go:build integration

package cdn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercdn-go/tools"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	cdnmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/cdn/mock"
)

const (
	testClientID    = 42
	testClientCname = "cl-42.edgecdn.ru"

	// data sources have no prior id, the read is what assigns one.
	unsetDataSourceID = "-"
)

func clientInfoReadCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Tools.On("ClientInfo", mock.Anything).
		Return(&tools.ClientInfoResponse{ID: testClientID, Cname: testClientCname}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "read client info",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, fmt.Sprintf("%d", testClientID))
			support.RequireStateAttrs(t, state, map[string]string{
				"cname":     testClientCname,
				"client_id": fmt.Sprintf("%d", testClientID),
			})
		},
	}
}

func clientInfoEmptyCnameCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Tools.On("ClientInfo", mock.Anything).
		Return(&tools.ClientInfoResponse{ID: testClientID, Cname: ""}, nil)

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "empty cname is an error",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "CNAME target is empty")
		},
	}
}

func clientInfoAPIFailureCase() support.ResourceCase[*cdnmock.MockedCDN] {
	mc := cdnmock.NewMockedCDN()

	mc.Tools.On("ClientInfo", mock.Anything).
		Return(nil, fmt.Errorf("api error: unauthorized"))

	return support.ResourceCase[*cdnmock.MockedCDN]{
		Name:         "API error on read",
		Op:           support.OpRead,
		Prepare:      func() *cdnmock.MockedCDN { return mc },
		CurrentID:    unsetDataSourceID,
		CurrentState: map[string]interface{}{},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cdnmock.MockedCDN) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "unauthorized")
		},
	}
}

func TestIntegrationClientInfo_TableDriven(t *testing.T) {
	t.Parallel()

	dataSource := cdnDataSource(t, "edgecenter_cdn_client_info")

	cases := []support.ResourceCase[*cdnmock.MockedCDN]{
		clientInfoReadCase(),
		clientInfoEmptyCnameCase(),
		clientInfoAPIFailureCase(),
	}

	support.RunResourceCases(t, dataSource, cases, support.DispatchCase[*cdnmock.MockedCDN])
}
