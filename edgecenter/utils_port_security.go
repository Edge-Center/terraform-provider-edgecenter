package edgecenter

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var ErrInstancePortSecNotImplemented = fmt.Errorf("instance_port_security are not impelemented yet")

func validatePortSecAttrs(d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var isPortSecDisabled, isSecGroupExists bool
	if v, ok := d.GetOk(PortSecurityDisabledField); ok {
		isPortSecDisabled = v.(bool)
	}
	_, isSecGroupExists = d.GetOk(SecurityGroupsField)
	if isPortSecDisabled && isSecGroupExists {
		curDiag := diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("if attribute \"%s\" set true, you can't set \"%s\" block", PortSecurityDisabledField, SecurityGroupsField),
			Detail:        "",
			AttributePath: nil,
		}
		diags = append(diags, curDiag)
	}

	return diags
}
