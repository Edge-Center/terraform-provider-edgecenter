package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func assignAllowedAddressPairs(ctx context.Context, client *edgecloudV2.Client, portID string, allowedAddressPairs []interface{}) *diag.Diagnostic {
	allowedAddressPairsRequest := edgecloudV2.PortsAllowedAddressPairsRequest{}
	for _, p := range allowedAddressPairs {
		pair := p.(map[string]interface{})
		allowedAddressPair := edgecloudV2.PortsAllowedAddressPairs{
			IPAddress:  pair["ip_address"].(string),
			MacAddress: pair["mac_address"].(string),
		}
		allowedAddressPairsRequest.AllowedAddressPairs = append(allowedAddressPairsRequest.AllowedAddressPairs, allowedAddressPair)
	}
	_, _, err := client.Ports.Assign(ctx, portID, &allowedAddressPairsRequest)
	if err != nil {
		return &diag.Diagnostic{Severity: diag.Warning, Summary: fmt.Sprintf("error from assigning allowed_address_pairs: %s ", err.Error())}
	}

	return nil
}
