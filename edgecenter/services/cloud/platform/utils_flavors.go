package platform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
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
		IncludePrices: d.Get(edgecenter.IncludePricesField).(bool),
	}
	flavors, _, err := client.Loadbalancers.FlavorList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch load balancer flavors: %w", err)
	}
	return flavors, nil
}

func newFlavorListOptions(d *schema.ResourceData) *edgecloudV2.FlavorListOptions {
	return &edgecloudV2.FlavorListOptions{
		IncludePrices:  d.Get(edgecenter.IncludePricesField).(bool),
		Disabled:       d.Get(edgecenter.IncludeDisabledField).(bool),
		ExcludeWindows: d.Get(edgecenter.ExcludeWindowsField).(bool),
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
		edgecenter.TypeField:                flavorType,
		edgecenter.FlavorIDField:            flavor.FlavorID,
		edgecenter.FlavorNameField:          flavor.FlavorName,
		edgecenter.RAMField:                 flavor.RAM,
		edgecenter.VCPUsField:               flavor.VCPUS,
		edgecenter.DisabledField:            flavor.Disabled,
		edgecenter.ResourceClassField:       flavor.ResourceClass,
		edgecenter.PricePerHourField:        flavor.PricePerHour,
		edgecenter.PricePerMonthField:       flavor.PricePerMonth,
		edgecenter.CurrencyCodeField:        flavor.CurrencyCode,
		edgecenter.HardwareDescriptionField: buildHardwareDescriptionMap(&flavor.HardwareDescription),
	}
}

func buildHardwareDescriptionMap(hw *edgecloudV2.HardwareDescription) map[string]interface{} {
	return map[string]interface{}{
		edgecenter.CPUField:         hw.CPU,
		edgecenter.IPUField:         hw.IPU,
		edgecenter.PoplarCountField: hw.PoplarCount,
		edgecenter.DiskField:        hw.Disk,
		edgecenter.NetworkField:     hw.Network,
		edgecenter.GPUField:         hw.GPU,
		edgecenter.RAMField:         hw.RAM,
		edgecenter.SgxEPCSizeField:  hw.SgxEPCSize,
	}
}
