package platform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	instanceFlavorType     = "instance"
	baremetalFlavorType    = "baremetal"
	loadBalancerFlavorType = "load_balancer"
	FlavorDataSource       = "edgecenter_flavor"
)

func dataSourceFlavor() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFlavorsRead,
		Description: "Represent flavors",
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			edgecenter.IncludeDisabledField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to include disabled flavors.",
			},
			edgecenter.ExcludeWindowsField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Set to true to exclude flavors dedicated for Windows images.",
			},
			edgecenter.IncludePricesField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Set to true if the response should include flavor prices. Default is true.",
			},
			edgecenter.TypeField: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Type of flavors to return: instance, baremetal, or load_balancer. If not specified, all flavors are returned.",
				ValidateFunc: validation.StringInSlice([]string{
					instanceFlavorType,
					baremetalFlavorType,
					loadBalancerFlavorType,
				}, false),
			},
			edgecenter.FlavorsField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of available flavors.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.TypeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor type: instance, baremetal, or load_balancer.",
						},
						edgecenter.FlavorIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor ID.",
						},
						edgecenter.FlavorNameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Flavor name.",
						},
						edgecenter.RAMField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "RAM size in MB.",
						},
						edgecenter.VCPUsField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of vCPUs.",
						},
						edgecenter.DisabledField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "The disabled flavor flag.",
						},
						edgecenter.ResourceClassField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The flavor resource class for mapping to hardware capacity.",
						},
						edgecenter.PricePerHourField: {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "The price per hour. Set if the include_prices query parameter is set to true",
						},
						edgecenter.PricePerMonthField: {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "The price per month. Set if the include_prices query parameter is set to true",
						},
						edgecenter.CurrencyCodeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The currency code. Set if the include_prices query parameter is set to true",
						},
						edgecenter.HardwareDescriptionField: {
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to initialize cloud client: %w", err))
	}

	typeFilter := d.Get(edgecenter.TypeField).(string)
	flavorOptions, err := fetchFlavorsForType(ctx, clientV2, d, typeFilter)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch flavors: %w", err))
	}

	resourceID := fmt.Sprintf("%d:%d", clientV2.Region, clientV2.Project)
	d.SetId(resourceID)

	if err := d.Set(edgecenter.FlavorsField, flavorOptions); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set flavors in state: %w", err))
	}

	tflog.Debug(ctx, "Finish flavor reading", map[string]interface{}{
		"flavor_count": len(flavorOptions),
		"resource_id":  resourceID,
	})

	return nil
}
