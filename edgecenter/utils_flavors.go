package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func fetchFlavorsForType(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData, typeFilter string) ([]interface{}, error) {
	showAll := typeFilter == ""

	var flavorOptions []interface{}

	if showAll || typeFilter == instanceFlavorType {
		flavors, err := fetchInstanceFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch instance flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, flavorsToInterface(flavors, instanceFlavorType)...)
	}

	if showAll || typeFilter == baremetalFlavorType {
		flavors, err := fetchBaremetalFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch baremetal flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, flavorsToInterface(flavors, baremetalFlavorType)...)
	}

	if showAll || typeFilter == loadBalancerFlavorType {
		flavors, err := fetchLoadBalancerFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch load balancer flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, flavorsToInterface(flavors, loadBalancerFlavorType)...)
	}

	return flavorOptions, nil
}

func fetchInstanceFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := newFlavorListOptions(d)
	flavors, _, err := client.Flavors.List(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance flavors: %w", err)
	}
	return flavors, nil
}

func fetchBaremetalFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := newFlavorListOptions(d)
	flavors, _, err := client.Flavors.ListBaremetal(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch baremetal flavors: %w", err)
	}
	return flavors, nil
}

func fetchLoadBalancerFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := &edgecloudV2.FlavorsOptions{
		IncludePrices: d.Get(IncludePricesField).(bool),
	}
	flavors, _, err := client.Loadbalancers.FlavorList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch load balancer flavors: %w", err)
	}
	return flavors, nil
}

func newFlavorListOptions(d *schema.ResourceData) *edgecloudV2.FlavorListOptions {
	return &edgecloudV2.FlavorListOptions{
		IncludePrices:  d.Get(IncludePricesField).(bool),
		Disabled:       d.Get(IncludeDisabledField).(bool),
		ExcludeWindows: d.Get(ExcludeWindowsField).(bool),
	}
}

func flavorsToInterface(flavors []edgecloudV2.Flavor, flavorType string) []interface{} {
	result := make([]interface{}, 0, len(flavors))
	for i := range flavors {
		result = append(result, flavorToMap(&flavors[i], flavorType))
	}
	return result
}

func flavorToMap(flavor *edgecloudV2.Flavor, flavorType string) map[string]interface{} {
	return map[string]interface{}{
		TypeField:                flavorType,
		FlavorIDField:            flavor.FlavorID,
		FlavorNameField:          flavor.FlavorName,
		RAMField:                 flavor.RAM,
		VCPUsField:               flavor.VCPUS,
		DisabledField:            flavor.Disabled,
		ResourceClassField:       flavor.ResourceClass,
		PricePerHourField:        flavor.PricePerHour,
		PricePerMonthField:       flavor.PricePerMonth,
		CurrencyCodeField:        flavor.CurrencyCode,
		HardwareDescriptionField: buildHardwareDescriptionMap(&flavor.HardwareDescription),
	}
}

func buildHardwareDescriptionMap(hw *edgecloudV2.HardwareDescription) map[string]interface{} {
	return map[string]interface{}{
		CPUField:         hw.CPU,
		IPUField:         hw.IPU,
		PoplarCountField: hw.PoplarCount,
		DiskField:        hw.Disk,
		NetworkField:     hw.Network,
		GPUField:         hw.GPU,
		RAMField:         hw.RAM,
		SgxEPCSizeField:  hw.SgxEPCSize,
	}
}
