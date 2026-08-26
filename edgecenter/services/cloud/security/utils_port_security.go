package security

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

var ErrInstancePortSecNotImplemented = fmt.Errorf("instance_port_security are not impelemented yet")

func validatePortSecAttrs(d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var isPortSecDisabled, isSecGroupExists bool
	if v, ok := d.GetOk(PortSecurityDisabledField); ok {
		isPortSecDisabled = v.(bool)
	}
	_, isSecGroupExists = d.GetOk(edgecenter.SecurityGroupsField)
	if isPortSecDisabled && isSecGroupExists {
		curDiag := diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("if attribute \"%s\" set true, you can't set \"%s\" block", PortSecurityDisabledField, edgecenter.SecurityGroupsField),
			Detail:        "",
			AttributePath: nil,
		}
		diags = append(diags, curDiag)
	}

	return diags
}

// removeSecurityGroupsFromInstancePort removes one or more security groups from a specific instance port.
func removeSecurityGroupsFromInstancePort(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, removeSGIDs []interface{}) error {
	if len(removeSGIDs) == 0 {
		return nil
	}
	sgsToRemove := make([]string, 0, len(removeSGIDs))
	for _, sg := range removeSGIDs {
		sgsToRemove = append(sgsToRemove, sg.(string))
	}
	removeSGOpts, err := PrepareAndValidateAssignSecurityGroupRequestOpts(ctx, client, sgsToRemove, portID)
	if err != nil {
		return err
	}
	_, err = client.Instances.SecurityGroupUnAssign(ctx, instanceID, removeSGOpts)
	if err != nil {
		return err
	}

	return nil
}

// AssignSecurityGroupsToInstancePort assigns one or more security groups to a specific instance port.
func AssignSecurityGroupsToInstancePort(ctx context.Context, client *edgecloudV2.Client, instanceID, portID string, assignSGIDs []interface{}) error {
	if len(assignSGIDs) == 0 {
		return nil
	}
	sgsToAssign := make([]string, 0, len(assignSGIDs))
	for _, sg := range assignSGIDs {
		sgsToAssign = append(sgsToAssign, sg.(string))
	}

	removeSGOpts, err := PrepareAndValidateAssignSecurityGroupRequestOpts(ctx, client, sgsToAssign, portID)
	if err != nil {
		return err
	}
	_, err = client.Instances.SecurityGroupAssign(ctx, instanceID, removeSGOpts)
	if err != nil {
		return err
	}

	return nil
}

func PrepareAndValidateAssignSecurityGroupRequestOpts(ctx context.Context, client *edgecloudV2.Client, sgIDs []string, portID string) (*edgecloudV2.AssignSecurityGroupRequest, error) {
	filteredSGs, err := utilV2.SecurityGroupListByIDs(ctx, client, sgIDs)
	if err != nil {
		return nil, err
	}

	sgsNames := make([]string, len(filteredSGs))
	for idx, sg := range filteredSGs {
		sgsNames[idx] = sg.Name
	}

	portSGNames := edgecloudV2.PortsSecurityGroupNames{
		SecurityGroupNames: sgsNames,
		PortID:             portID,
	}

	sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

	return &sgOpts, nil
}
