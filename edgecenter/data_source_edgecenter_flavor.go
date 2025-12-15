package edgecenter

import (
	"context"
	"fmt"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	instanceFlavorType     = "instance"
	baremetalFlavorType    = "baremetal"
	loadBalancerFlavorType = "load_balancer"
)

func dataSourceFlavor() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFlavorsRead,
		Description: "Represent flavors",
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			IncludeDisabledField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to include disabled flavors.",
			},
			ExcludeWindowsField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to exclude flavors dedicated for Windows images.",
			},
			IncludePricesField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true if the response should include flavor prices.",
			},
			TypeField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Type of flavors to return: instance, baremetal, or load_balancer. If not specified, all flavors are returned.",
				ValidateFunc: validation.StringInSlice([]string{
					instanceFlavorType,
					baremetalFlavorType,
					loadBalancerFlavorType,
				}, false),
			},
			FlavorsField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of available flavors.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor type: instance, baremetal, or load_balancer.",
						},
						FlavorIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor ID.",
						},
						FlavorNameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor name.",
						},
						RAMField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "RAM size in MB.",
						},
						VCPUsField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of vCPUs.",
						},
						DisabledField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "The disabled flavor flag.",
						},
						ResourceClassField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The flavor resource class for mapping to hardware capacity.",
						},
						PricePerHourField: {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "The price per hour. Set if the include_prices query parameter is set to true",
						},
						PricePerMonthField: {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "The price per month. Set if the include_prices query parameter is set to true",
						},
						CurrencyCodeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The currency code. Set if the include_prices query parameter is set to true",
						},
						HardwareDescriptionField: {
							Type:        schema.TypeMap,
							Computed:    true,
							Description: "An additional hardware description.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceFlavorsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start flavor reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to initialize cloud client: %w", err))
	}

	typeFilter := d.Get(TypeField).(string)
	flavorOptions, err := fetchFlavorsForType(ctx, clientV2, d, typeFilter)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch flavors: %w", err))
	}

	resourceID := fmt.Sprintf("%d:%d", clientV2.Region, clientV2.Project)
	d.SetId(resourceID)

	if err := d.Set(FlavorsField, flavorOptions); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set flavors in state: %w", err))
	}

	tflog.Debug(ctx, "Finish flavor reading", map[string]interface{}{
		"flavor_count": len(flavorOptions),
		"resource_id":  resourceID,
	})

	return nil
}

func fetchFlavorsForType(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData, typeFilter string) ([]interface{}, error) {
	showAll := typeFilter == ""

	var flavorOptions []interface{}

	if showAll || typeFilter == instanceFlavorType {
		flavors, err := fetchInstanceFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch instance flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, prepareFlavors(flavors, instanceFlavorType)...)
	}

	if showAll || typeFilter == baremetalFlavorType {
		flavors, err := fetchBaremetalFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch baremetal flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, prepareFlavors(flavors, baremetalFlavorType)...)
	}

	if showAll || typeFilter == loadBalancerFlavorType {
		flavors, err := fetchLoadBalancerFlavors(ctx, client, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch load balancer flavors: %w", err)
		}
		flavorOptions = append(flavorOptions, prepareFlavors(flavors, loadBalancerFlavorType)...)
	}

	return flavorOptions, nil
}

func fetchInstanceFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := buildFlavorListOptions(d)
	flavors, _, err := client.Flavors.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return flavors, nil
}

func fetchBaremetalFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := buildFlavorListOptions(d)
	flavors, _, err := client.Flavors.ListBaremetal(ctx, options)
	if err != nil {
		return nil, err
	}
	return flavors, nil
}

func fetchLoadBalancerFlavors(ctx context.Context, client *edgecloudV2.Client, d *schema.ResourceData) ([]edgecloudV2.Flavor, error) {
	options := &edgecloudV2.FlavorsOptions{
		IncludePrices: d.Get(IncludePricesField).(bool),
	}
	flavors, _, err := client.Loadbalancers.FlavorList(ctx, options)
	if err != nil {
		return nil, err
	}
	return flavors, nil
}

func buildFlavorListOptions(d *schema.ResourceData) *edgecloudV2.FlavorListOptions {
	return &edgecloudV2.FlavorListOptions{
		IncludePrices:  d.Get(IncludePricesField).(bool),
		Disabled:       d.Get(IncludeDisabledField).(bool),
		ExcludeWindows: d.Get(ExcludeWindowsField).(bool),
	}
}

func prepareFlavors(flavors []edgecloudV2.Flavor, flavorType string) []interface{} {
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
