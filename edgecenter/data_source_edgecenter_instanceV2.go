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
		Description: `A cloud instance is a virtual machine in a cloud environment. Could be used with baremetal also.`,
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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the instance.",
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
			InstanceInterfaceField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NetworkIDField: {
							Type:     schema.TypeString,
							Computed: true,
						},
						SubnetIDField: {
							Type:     schema.TypeString,
							Computed: true,
						},
						PortIDField: {
							Type:     schema.TypeString,
							Computed: true,
						},
						IPAddressField: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			SecurityGroupField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of firewall configurations applied to the instance, defined by their id and name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NameField: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			MetadataField: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						KeyField: {
							Type:     schema.TypeString,
							Computed: true,
						},
						ValueField: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
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
			InstanceAddressesField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of network addresses associated with the instance, for example "pub_net": [...].`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						InstanceAddressesNetField: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									InstanceAddressesAddrField: {
										Type:     schema.TypeString,
										Computed: true,
									},
									TypeField: {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceInstanceV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get(NameField).(string)

	insts, _, err := clientV2.Instances.List(ctx, &edgecloudV2.InstanceListOptions{Name: name})
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var instance edgecloudV2.Instance
	for _, l := range insts {
		if l.Name == name {
			instance = l
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("instance with name %s not found", name)
	}

	d.SetId(instance.ID)
	d.Set(NameField, instance.Name)
	d.Set(FlavorIDField, instance.Flavor.FlavorID)
	d.Set(StatusField, instance.Status)
	d.Set(InstanceVMStateField, instance.VMState)

	flavor := make(map[string]interface{}, 4)
	flavor[FlavorIDField] = instance.Flavor.FlavorID
	flavor[FlavorNameField] = instance.Flavor.FlavorName
	flavor[RAMField] = strconv.Itoa(instance.Flavor.RAM)
	flavor[VCPUsField] = strconv.Itoa(instance.Flavor.VCPUS)
	d.Set(FlavorField, flavor)

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
	log.Printf("instance data source interfaces: %+v", ifs)
	if err != nil {
		return diag.FromErr(err)
	}
	var cleanInterfaces []interface{}
	for _, iface := range ifs {
		if len(iface.IPAssignments) == 0 {
			continue
		}

		for _, assignment := range iface.IPAssignments {
			subnetID := assignment.SubnetID

			i := make(map[string]interface{})

			i[NetworkIDField] = iface.NetworkID
			i[SubnetIDField] = subnetID
			i[PortIDField] = iface.PortID
			i[IPAddressField] = iface.IPAssignments[0].IPAddress.String()

			cleanInterfaces = append(cleanInterfaces, i)
		}
	}
	if err := d.Set(InstanceInterfaceField, cleanInterfaces); err != nil {
		return diag.FromErr(err)
	}

	sliced := make([]map[string]interface{}, 0, len(instance.Metadata))
	for k, data := range instance.Metadata {
		mdata := make(map[string]interface{}, 2)
		mdata[KeyField] = k
		mdata[ValueField] = data
		sliced = append(sliced, mdata)
	}
	if err := d.Set(MetadataField, sliced); err != nil {
		return diag.FromErr(err)
	}

	secGrps := make([]map[string]interface{}, 0, len(instance.SecurityGroups))
	for _, sg := range instance.SecurityGroups {
		i := make(map[string]interface{})
		i[NameField] = sg.Name
		secGrps = append(secGrps, i)
	}
	if err := d.Set(SecurityGroupField, secGrps); err != nil {
		return diag.FromErr(err)
	}

	addresses := []map[string][]map[string]string{}
	for _, data := range instance.Addresses {
		d := map[string][]map[string]string{}
		netd := make([]map[string]string, len(data))
		for i, iaddr := range data {
			ndata := make(map[string]string, 2)
			ndata[TypeField] = iaddr.Type
			ndata[InstanceAddressesAddrField] = iaddr.Address.String()
			netd[i] = ndata
		}
		d[InstanceAddressesNetField] = netd
		addresses = append(addresses, d)
	}
	if err := d.Set(InstanceAddressesField, addresses); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Instance reading")

	return diags
}
