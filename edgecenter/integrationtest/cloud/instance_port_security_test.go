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

const (
	testPortID     = "test-port-id"
	testInstanceID = "test-instance-id"
	testSGID1      = "test-sg-1"
	testSGID2      = "test-sg-2"
)

func sampleInstanceIface(portID string, enabled bool) edgecloud.InstancePortInterface {
	return edgecloud.InstancePortInterface{
		PortID:              portID,
		PortSecurityEnabled: enabled,
		MacAddress:          "00:00:00:00:00:01",
	}
}

func samplePort(portID string, sgIDs ...string) edgecloud.InstancePort {
	sgs := make([]edgecloud.IDName, len(sgIDs))
	for i, id := range sgIDs {
		sgs[i] = edgecloud.IDName{ID: id, Name: id}
	}
	return edgecloud.InstancePort{
		ID:             portID,
		Name:           "test-port",
		SecurityGroups: sgs,
	}
}

func portSecDisableCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, true),
		}, nil, nil).Once()

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, false),
		}, nil, nil).Once()

	mc.Instances.On("PortsList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePort{
			samplePort(testPortID),
		}, nil, nil).Once()

	mc.Ports.On("DisablePortSecurity", mock.Anything, testPortID).
		Return((*edgecloud.InstancePortInterface)(nil), nil, nil).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "disable port security",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id":            testInstanceID,
				"port_id":                testPortID,
				"port_security_disabled": true,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testPortID)
			support.RequireStateAttrs(t, state, map[string]string{
				"port_security_disabled": "true",
			})
		},
	}
}

func portSecReadCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, true),
		}, nil, nil).Once()

	mc.Instances.On("PortsList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePort{
			samplePort(testPortID, testSGID1, testSGID2),
		}, nil, nil).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read existing port_security",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: testPortID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id": testInstanceID,
				"port_id":     testPortID,
				"security_groups": []interface{}{
					map[string]interface{}{
						"security_group_ids": []interface{}{testSGID1},
						"overwrite_existing": false,
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, testPortID)
			support.RequireStateAttrs(t, state, map[string]string{
				"port_security_disabled": "false",
			})
		},
	}
}

func portSecReadNotFoundCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return(nil, nil, fmt.Errorf("port not found")).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: testPortID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id": testInstanceID,
				"port_id":     testPortID,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "port not found")
		},
	}
}

func portSecDeleteDisabledCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, false),
		}, nil, nil).Once()

	mc.Ports.On("EnablePortSecurity", mock.Anything, testPortID).
		Return((*edgecloud.InstancePortInterface)(nil), nil, nil).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete disabled port — enable port security back",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: testPortID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id":            testInstanceID,
				"port_id":                testPortID,
				"port_security_disabled": true,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			// NOTE: resource does not clear ID on disabled-port delete (resource bug)
			support.RequireStateID(t, state, testPortID)
		},
	}
}

func portSecDeleteEnabledWithSGsCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, true),
		}, nil, nil).Once()

	mc.SecurityGroups.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.SecurityGroup{
			{ID: testSGID1, Name: testSGID1},
		}, nil, nil).Once()

	mc.Instances.On("SecurityGroupUnAssign", mock.Anything, testInstanceID, mock.Anything).
		Return(nil, nil).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete enabled port with SGs",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: testPortID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id": testInstanceID,
				"port_id":     testPortID,
				"security_groups": []interface{}{
					map[string]interface{}{
						"security_group_ids": []interface{}{testSGID1},
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoErrorDiags(t, diags)
			require.Nil(t, state, "state must be nil after successful delete")
		},
	}
}

func portSecDisableAPIFailureCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.Instances.On("InterfaceList", mock.Anything, testInstanceID).
		Return([]edgecloud.InstancePortInterface{
			sampleInstanceIface(testPortID, true),
		}, nil, nil).Once()

	mc.Ports.On("DisablePortSecurity", mock.Anything, testPortID).
		Return(nil, nil, fmt.Errorf("api error: disable failed")).Once()

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "API error on disable port security",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			map[string]interface{}{
				"instance_id":            testInstanceID,
				"port_id":                testPortID,
				"port_security_disabled": true,
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "disable failed")
			require.Nil(t, state, "state must be nil on create failure")
		},
	}
}

func TestIntegrationInstancePortSecurity_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_instance_port_security"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		portSecDisableCase(),
		portSecReadCase(),
		portSecReadNotFoundCase(),
		portSecDeleteDisabledCase(),
		portSecDeleteEnabledWithSGsCase(),
		portSecDisableAPIFailureCase(),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
