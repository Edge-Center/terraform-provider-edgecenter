package edgecenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func allocationPoolsToListOfMaps(allocationPools []edgecloudV2.AllocationPool) []interface{} {
	aps := make([]interface{}, 0, len(allocationPools))

	for _, v := range allocationPools {
		ap := map[string]interface{}{StartField: "", EndField: ""}
		ap[StartField], ap[EndField] = v.Start, v.End
		aps = append(aps, ap)
	}

	return aps
}

// getSubnet retrieves a subnet from the edge cloud service.
// It attempts to find the subnet either by its ID or by its name.
func getSubnet(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Subnetwork, error) {
	var (
		subnet *edgecloudV2.Subnetwork
		err    error
	)

	name := d.Get(NameField).(string)
	networkID := d.Get(NetworkIDField).(string)
	subnetID := d.Get(IDField).(string)

	switch {
	case subnetID != "":
		subnet, _, err = clientV2.Subnetworks.Get(ctx, subnetID)
		if err != nil {
			return nil, err
		}
	default:
		subnetsOpts := &edgecloudV2.SubnetworkListOptions{NetworkID: networkID}

		if metadataK, ok := d.GetOk(MetadataKField); ok {
			subnetsOpts.MetadataK = metadataK.(string)
		}
		if metadataRaw, ok := d.GetOk(MetadataKVField); ok {
			typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
			for k, v := range metadataRaw.(map[string]interface{}) {
				typedMetadataKV[k] = v.(string)
			}
			typedMetadataKVJson, err := json.Marshal(typedMetadataKV)
			if err != nil {
				return nil, err
			}
			subnetsOpts.MetadataKV = string(typedMetadataKVJson)
		}

		snets, _, err := clientV2.Subnetworks.List(ctx, subnetsOpts)
		if err != nil {
			return nil, err
		}

		foundSubnets := make([]edgecloudV2.Subnetwork, 0, len(snets))

		for _, sn := range snets {
			if sn.Name == name {
				foundSubnets = append(foundSubnets, sn)
			}
		}

		switch {
		case len(foundSubnets) == 0:
			return nil, errors.New("subnet does not exist")

		case len(foundSubnets) > 1:
			var message bytes.Buffer
			message.WriteString("Found subnets:\n")

			for _, snet := range foundSubnets {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", snet.ID))
			}

			return nil, fmt.Errorf("multiple subnets found.\n %s.\n Use subnet ID instead of name", message.String())
		}

		subnet = &snets[0]
	}

	return subnet, nil
}
