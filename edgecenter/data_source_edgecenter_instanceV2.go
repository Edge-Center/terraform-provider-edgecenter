package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

var volumeElemResource = schema.Resource{
	Schema: map[string]*schema.Schema{
		NameField: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name assigned to the volume. Defaults to 'system'.",
		},
		TypeNameField: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The type of volume to create. Valid values are 'ssd_hiiops', 'standard', 'cold', and 'ultra'. Defaults to 'standard'.",
		},
		InstanceVolumeSizeField: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The size of the volume, specified in gigabytes (GB).",
		},
		InstanceVolumeIDField: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of the volume.",
		},
	},
}

func dataSourceInstanceV2() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceInstanceV2Read,
		Description: `A cloud instance is a virtual machine in a cloud environment. Could be used with baremetal too.`,
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the instance. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			IDField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the instance. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			FlavorIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the flavor to be used for the instance, determining its compute and memory, for example 'g1-standard-2-4'.",
			},
			InstanceBootVolumesField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem:        &volumeElemResource,
			},
			InstanceDataVolumesField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem:        &volumeElemResource,
			},
			AvailabilityZoneField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The availability zone where the instance is located.",
			},
			InstanceInterfacesField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NetworkIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the network.",
						},
						NetworkNameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the network.",
						},
						OrderField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Order of attaching interface.",
						},
						SubnetIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the subnet.",
						},
						PortIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID ot the port.",
						},
						IPAddressField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP address of the interface.",
						},
					},
				},
			},
			MetadataField: {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			FlavorField: {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: `A map defining the flavor of the instance, for example, {"flavor_name": "g1-standard-2-4", "ram": 4096, ...}.`,
			},
			StatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the instance. This is computed automatically and can be used to track the instance's state.",
			},
			InstanceVMStateField: {
				Type:     schema.TypeString,
				Computed: true,
				Description: fmt.Sprintf(`The current virtual machine state of the instance, 
allowing you to start or stop the VM. Possible values are %s and %s.`, InstanceVMStateStopped, InstanceVMStateActive),
			},
		},
	}
}

func dataSourceInstanceV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := getInstance(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(instance.ID)
	_ = d.Set(NameField, instance.Name)
	_ = d.Set(IDField, instance.ID)
	_ = d.Set(FlavorIDField, instance.Flavor.FlavorID)
	_ = d.Set(StatusField, instance.Status)
	_ = d.Set(InstanceVMStateField, instance.VMState)
	_ = d.Set(AvailabilityZoneField, instance.AvailabilityZone)

	flavor := make(map[string]interface{}, 4)
	flavor[FlavorIDField] = instance.Flavor.FlavorID
	flavor[FlavorNameField] = instance.Flavor.FlavorName
	flavor[RAMField] = strconv.Itoa(instance.Flavor.RAM)
	flavor[VCPUsField] = strconv.Itoa(instance.Flavor.VCPUS)
	_ = d.Set(FlavorField, flavor)

	volumesReq := edgecloudV2.VolumeListOptions{
		InstanceID: instance.ID,
	}

	instanceVolumes, _, err := clientV2.Volumes.List(ctx, &volumesReq)
	if err != nil {
		return diag.FromErr(err)
	}

	bootVolumesData, dataVolumesData := PrepareVolumesDataToSet(instanceVolumes)
	if err := d.Set(InstanceBootVolumesField, bootVolumesData); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(InstanceDataVolumesField, dataVolumesData); err != nil {
		return diag.FromErr(err)
	}

	ifs, _, err := clientV2.Instances.InterfaceList(ctx, instance.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("instance data source interfaces: %+v", ifs)

	var cleanInterfaces []interface{}
	for index, iface := range ifs {
		if len(iface.IPAssignments) == 0 {
			continue
		}

		for _, assignment := range iface.IPAssignments {
			subnetID := assignment.SubnetID

			i := make(map[string]interface{})

			i[NetworkIDField] = iface.NetworkID
			i[SubnetIDField] = subnetID
			i[OrderField] = index + 1
			i[PortIDField] = iface.PortID
			i[NetworkNameField] = iface.NetworkDetails.Name
			i[IPAddressField] = iface.IPAssignments[0].IPAddress.String()

			cleanInterfaces = append(cleanInterfaces, i)
		}
	}

	if err := d.Set(InstanceInterfacesField, cleanInterfaces); err != nil {
		return diag.FromErr(err)
	}

	metadata := make(map[string]interface{}, len(instance.Metadata))
	for key, value := range instance.Metadata {
		metadata[key] = value
	}
	if err = d.Set(MetadataField, metadata); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Instance reading")

	return nil
}
