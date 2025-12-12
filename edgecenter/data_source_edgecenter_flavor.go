package edgecenter

import (
	"context"
	"strconv"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			FlavorsField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of available flavors (VM and baremetal).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
		return diag.FromErr(err)
	}

	flavors, _, err := clientV2.Flavors.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	baremetalFlavors, _, err := clientV2.Flavors.ListBaremetal(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	flavors = append(flavors, baremetalFlavors...)

	d.SetId(strconv.Itoa(clientV2.Region))

	flavorOptions := prepareFlavors(flavors)
	if err := d.Set("flavors", flavorOptions); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish flavor reading")

	return nil
}

func prepareFlavors(flavors []edgecloudV2.Flavor) []interface{} {
	result := make([]interface{}, 0, len(flavors))
	for _, flavor := range flavors {
		result = append(result, map[string]interface{}{
			FlavorIDField:      flavor.FlavorID,
			FlavorNameField:    flavor.FlavorName,
			RAMField:           flavor.RAM,
			VCPUsField:         flavor.VCPUS,
			DisabledField:      flavor.Disabled,
			ResourceClassField: flavor.ResourceClass,
			HardwareDescriptionField: map[string]interface{}{
				CPUField:         flavor.HardwareDescription.CPU,
				IPUField:         flavor.HardwareDescription.IPU,
				PoplarCountField: flavor.HardwareDescription.PoplarCount,
				DiskField:        flavor.HardwareDescription.Disk,
				NetworkField:     flavor.HardwareDescription.Network,
				GPUField:         flavor.HardwareDescription.GPU,
				RAMField:         flavor.HardwareDescription.RAM,
				SgxEpcSizeField:  flavor.HardwareDescription.SgxEPCSize,
			},
		})
	}
	return result
}
