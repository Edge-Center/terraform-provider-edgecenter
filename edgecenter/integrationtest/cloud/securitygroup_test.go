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

const testSecurityGroupID = "sg-id"

func sampleSecurityGroup(id, name string) *edgecloud.SecurityGroup {
	return &edgecloud.SecurityGroup{
		ID:          id,
		Name:        name,
		Description: "test security group",
		ProjectID:   testProjectID,
		RegionID:    testRegionID,
		Metadata:    []edgecloud.MetadataDetailed{},
		SecurityGroupRules: []edgecloud.SecurityGroupRule{
			{
				ID:        "rule-ingress",
				Direction: edgecloud.SGRuleDirectionIngress,
				Protocol:  &tcpProto,
				EtherType: &ipv4,
				PortRangeMax:  intPtr(22),
				PortRangeMin:  intPtr(22),
				Description:   strPtr("SSH"),
				RemoteIPPrefix: strPtr("0.0.0.0/0"),
			},
			{
				ID:        "rule-egress",
				Direction: edgecloud.SGRuleDirectionEgress,
				Protocol:  &anyProto,
				EtherType: &ipv4,
				PortRangeMax:  nil,
				PortRangeMin:  nil,
				Description:   strPtr(""),
				RemoteIPPrefix: strPtr(""),
			},
		},
	}
}

var (
	tcpProto  = edgecloud.SGRuleProtocolTCP
	anyProto  = edgecloud.SGRuleProtocolANY
	ipv4      = edgecloud.EtherTypeIPv4
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int { return &i }

func securityGroupCreateWithRulesCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.SecurityGroups.On("Create", mock.Anything,
		mock.MatchedBy(func(req *edgecloud.SecurityGroupCreateRequest) bool {
			return req.SecurityGroup.Name == "test-sg"
		}),
	).Return(sampleSecurityGroup(sgID, "test-sg"), nil, nil)

	mc.SecurityGroups.On("Get", mock.Anything, sgID).
		Return(sampleSecurityGroup(sgID, "test-sg"), nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "create with rules",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, sgID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "test-sg",
			})
		},
	}
}

func securityGroupUpdateNameCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.SecurityGroups.On("Update", mock.Anything, sgID,
		mock.MatchedBy(func(req *edgecloud.SecurityGroupUpdateRequest) bool {
			return req.Name == "new-name"
		}),
	).Return((*edgecloud.SecurityGroup)(nil), nil, nil)

	updated := sampleSecurityGroup(sgID, "new-name")
	mc.SecurityGroups.On("Get", mock.Anything, sgID).
		Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update name",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("new-name"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, sgID)
			support.RequireStateAttrs(t, state, map[string]string{
				"name": "new-name",
			})
		},
	}
}

func securityGroupAddRemoveRuleCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	mc.SecurityGroups.On("RuleCreate", mock.Anything, sgID,
		mock.MatchedBy(func(req *edgecloud.RuleCreateRequest) bool {
			return req.Direction == edgecloud.SGRuleDirectionIngress && req.Protocol == edgecloud.SGRuleProtocolTCP
		}),
	).Return(&edgecloud.SecurityGroupRule{
		ID:        "rule-new",
		Direction: edgecloud.SGRuleDirectionIngress,
		Protocol:  &tcpProto,
		EtherType: &ipv4,
	}, nil, nil)

	mc.SecurityGroups.On("RuleDelete", mock.Anything, "rule-ingress").
		Return(&edgecloud.TaskResponse{}, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNoContent}}, nil)

	updated := sampleSecurityGroup(sgID, "test-sg")
	updated.SecurityGroupRules = []edgecloud.SecurityGroupRule{
		{
			ID:        "rule-new",
			Direction: edgecloud.SGRuleDirectionIngress,
			Protocol:  &tcpProto,
			EtherType: &ipv4,
			PortRangeMax:  intPtr(80),
			PortRangeMin:  intPtr(80),
			Description:   strPtr("HTTP"),
		},
		{
			ID:        "rule-old",
			Direction: edgecloud.SGRuleDirectionEgress,
			Protocol:  &anyProto,
			EtherType: &ipv4,
		},
	}

	mc.SecurityGroups.On("Get", mock.Anything, sgID).
		Return(updated, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "add and remove rule",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"id":          "rule-egress",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
						"id":               "rule-ingress",
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"id":          "rule-old",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   80,
						"port_range_max":   80,
						"description":      "HTTP",
						"remote_ip_prefix": "",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, sgID)
		},
	}
}

func securityGroupDeleteCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.SecurityGroups.On("Delete", mock.Anything, sgID).
		Return(nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "delete security group",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
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

func securityGroupValidationNoEgressCase() support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.AllowProjectResolution(mc, testProjectID)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:    "validation: no egress rule",
		Op:      support.OpApply,
		Prepare: func() *cloudmock.MockedCloud { return mc },
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "ingress",
						"ethertype":   "IPv4",
						"protocol":    "tcp",
						"port_range_min": 22,
						"port_range_max": 22,
						"description": "SSH",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "at least one 'egress' rule should be set")
			require.Nil(t, state, "state must be nil when validation fails")
		},
	}
}

func securityGroupReadNonExistentCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.SecurityGroups.On("Get", mock.Anything, sgID).
		Return(nil, &edgecloud.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("not found"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "read non-existent (404)",
		Op:        support.OpRead,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "not found")
			require.NotNil(t, state, "state must not be cleared when read fails")
			require.Equal(t, sgID, state.ID)
		},
	}
}

func securityGroupDeleteAPIFailureCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 1)

	mc.SecurityGroups.On("Delete", mock.Anything, sgID).
		Return(&edgecloud.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, fmt.Errorf("api error"))

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "API error on delete",
		Op:        support.OpDelete,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireHasErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "api error")
			require.NotNil(t, state, "state must not be nil when delete fails")
			require.Equal(t, sgID, state.ID, "ID must not be cleared on failed delete")
		},
	}
}

func securityGroupMetadataUpdateCase(sgID string) support.ResourceCase[*cloudmock.MockedCloud] {
	mc := cloudmock.NewMockedCloud(testProjectID, testRegionID)
	cloudmock.ExpectProjectResolutionTimes(mc, testProjectID, 2)

	sgWithMeta := sampleSecurityGroup(sgID, "test-sg")
	sgWithMeta.Metadata = []edgecloud.MetadataDetailed{
		{Key: "env", Value: "prod", ReadOnly: false},
	}

	mc.SecurityGroups.On("MetadataUpdate", mock.Anything, sgID,
		mock.MatchedBy(func(meta *edgecloud.Metadata) bool {
			return len(*meta) == 1 && (*meta)["env"] == "prod"
		}),
	).Return(nil, nil)

	mc.SecurityGroups.On("Get", mock.Anything, sgID).
		Return(sgWithMeta, nil, nil)

	return support.ResourceCase[*cloudmock.MockedCloud]{
		Name:      "update metadata",
		Op:        support.OpApply,
		Prepare:   func() *cloudmock.MockedCloud { return mc },
		CurrentID: sgID,
		CurrentState: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
					},
				},
			},
		),
		NewConfig: cloud.Merge(
			cloud.WithProjectRegion(testProjectID, testRegionID),
			cloud.WithName("test-sg"),
			cloud.WithMetadata(map[string]string{"env": "prod"}),
			map[string]interface{}{
				"description": "test security group",
				"security_group_rules": []interface{}{
					map[string]interface{}{
						"direction":   "egress",
						"ethertype":   "IPv4",
						"protocol":    "any",
						"description": "",
					},
					map[string]interface{}{
						"direction":        "ingress",
						"ethertype":        "IPv4",
						"protocol":         "tcp",
						"port_range_min":   22,
						"port_range_max":   22,
						"description":      "SSH",
						"remote_ip_prefix": "0.0.0.0/0",
					},
				},
			},
		),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *cloudmock.MockedCloud) {
			support.RequireNoDiags(t, diags)
			support.RequireStateID(t, state, sgID)
		},
	}
}

func TestIntegrationSecurityGroup_TableDriven(t *testing.T) {
	t.Parallel()

	resource := edgecenter.Provider().ResourcesMap["edgecenter_securitygroup"]

	cases := []support.ResourceCase[*cloudmock.MockedCloud]{
		securityGroupCreateWithRulesCase(testSecurityGroupID),
		securityGroupUpdateNameCase(testSecurityGroupID),
		securityGroupAddRemoveRuleCase(testSecurityGroupID),
		securityGroupMetadataUpdateCase(testSecurityGroupID),
		securityGroupDeleteCase(testSecurityGroupID),
		securityGroupValidationNoEgressCase(),
		securityGroupReadNonExistentCase(testSecurityGroupID),
		securityGroupDeleteAPIFailureCase(testSecurityGroupID),
	}

	support.RunResourceCases(t, resource, cases, support.DispatchCase[*cloudmock.MockedCloud])
}
