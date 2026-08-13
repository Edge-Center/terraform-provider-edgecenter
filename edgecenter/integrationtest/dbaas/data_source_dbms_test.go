//go:build integration

package dbaas_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	dbaasmock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/dbaas/mock"
	dbaassvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/dbaas"
)

func TestIntegrationDBaaSDbmsDataSource_Read(t *testing.T) {
	t.Parallel()

	dataSource := dbaasDataSource(t, dbaassvc.DBaaSDbmsDataSource)

	cases := []support.ResourceCase[*dbaasmock.MockedDBaaS]{
		{
			Name: "read lists every engine and keys the data source by the project",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DbmsList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSDbms{
						{ID: 1, Type: "POSTGRESQL", Version: "16.9"},
						{ID: 2, Type: "POSTGRESQL", Version: "17.5"},
					}, nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: projectRegion(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testProjectIDStr)
				support.RequireStateAttrs(t, state, map[string]string{
					"items.#":         "2",
					"items.0.id":      "1",
					"items.0.type":    "POSTGRESQL",
					"items.0.version": "16.9",
					"items.1.version": "17.5",
				})
			},
		},
		{
			Name: "an empty engine list is not an error",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DbmsList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSDbms{}, nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: projectRegion(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateAttrs(t, state, map[string]string{"items.#": "0"})
			},
		},
		{
			Name: "read surfaces the api error",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DbmsList", mock.Anything, mock.Anything).
					Return(nil, statusResponse(500), fmt.Errorf("api error: backend is down")).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: projectRegion(),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireOnlyErrorDiags(t, diags)
				support.RequireErrorDiagContains(t, diags, "backend is down")
			},
		},
		{
			Name: "the id ignores the region, so two regions of one project collide",
			Op:   support.OpRead,
			Prepare: func() *dbaasmock.MockedDBaaS {
				mc := dbaasmock.NewMockedDBaaS()
				dbaasmock.AllowProjectResolution(mc, testProjectID)

				mc.DBaaS.On("DbmsList", mock.Anything, mock.Anything).
					Return([]edgecloud.DBaaSDbms{{ID: 1, Type: "POSTGRESQL", Version: "17.5"}}, nil, nil).Once()

				return mc
			},
			CurrentID:    unsetDataSourceID,
			CurrentState: merge(projectRegion(), map[string]interface{}{"region_id": 11}),
			Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *dbaasmock.MockedDBaaS) {
				support.RequireNoErrorDiags(t, diags)
				support.RequireStateID(t, state, testProjectIDStr)
				require.Equal(t, "11", state.Attributes["region_id"])
			},
		},
	}

	support.RunResourceCases(t, dataSource, cases, runDataSourceRead)
}
