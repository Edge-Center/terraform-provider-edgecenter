package edgecenter

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func validateSubnetGatewayIP(i interface{}, k string) ([]string, []error) {
	if i.(string) == disable {
		return nil, nil
	}

	return validation.IsIPAddress(i, k)
}

func prepareSubnetAllocationPools(aps []interface{}) []edgecloudV2.AllocationPool {
	allocationPools := make([]edgecloudV2.AllocationPool, 0, len(aps))

	for _, v := range aps {
		ap := v.(map[string]interface{})
		allocationPools = append(allocationPools, edgecloudV2.AllocationPool{
			Start: ap[StartField].(string),
			End:   ap[EndField].(string),
		})
	}

	return allocationPools
}

func allocationPoolsToListOfMaps(allocationPools []edgecloudV2.AllocationPool) []map[string]string {
	aps := make([]map[string]string, 0, len(allocationPools))

	for _, v := range allocationPools {
		ap := map[string]string{StartField: "", EndField: ""}
		ap[StartField], ap[EndField] = v.Start, v.End
		aps = append(aps, ap)
	}

	return aps
}
