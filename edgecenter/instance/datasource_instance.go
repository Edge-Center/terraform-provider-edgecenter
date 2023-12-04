package instance

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterInstance() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterInstanceRead,
		Description: `A cloud instance is a virtual machine in a cloud environment`,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the project",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the region",
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "instance uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Description: `instance name. this parameter is not unique, if there is more than one instance with the same name, 
then the first one will be used. it is recommended to use "id"`,
				ExactlyOneOf: []string{"id", "name"},
			},
			// computed attributes
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "current status of the instance resource",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "name of the region",
			},
			"vm_state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "state of the virtual machine",
			},
			"keypair_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "name of the keypair",
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "metadata in detailed format",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"volumes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of volumes ID's",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"security_groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of security groups names",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"flavor": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "information about the flavor",
			},
			"addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "network addresses associated with the instance",
				Elem:        &schema.Schema{Type: schema.TypeMap},
			},
			"interface": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "network interfaces attached to the instance",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceEdgeCenterInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundInstance *edgecloud.Instance

	if id, ok := d.GetOk("id"); ok {
		instance, _, err := client.Instances.Get(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundInstance = instance
	} else if instanceName, ok := d.GetOk("name"); ok {
		instList, _, err := client.Instances.List(ctx, &edgecloud.InstanceListOptions{Name: instanceName.(string)})
		if err != nil {
			return diag.FromErr(err)
		}

		foundInstance = &instList[0]
	} else {
		return diag.Errorf("Error: specify either id or a name to lookup the instance")
	}

	d.SetId(foundInstance.ID)
	d.Set("name", foundInstance.Name)

	ifs, _, err := client.Instances.InterfaceList(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var cleanInterfaces []interface{}
	for _, iface := range ifs {
		if len(iface.IPAssignments) == 0 {
			continue
		}

		for _, assignment := range iface.IPAssignments {
			i := map[string]interface{}{
				"network_id": iface.NetworkID,
				"subnet_id":  assignment.SubnetID,
				"port_id":    iface.PortID,
				"ip_address": iface.IPAssignments[0].IPAddress.String(),
			}
			cleanInterfaces = append(cleanInterfaces, i)
		}
	}
	if err := d.Set("interface", cleanInterfaces); err != nil {
		return diag.FromErr(err)
	}

	d.Set("status", foundInstance.Status)
	d.Set("region", foundInstance.Region)
	d.Set("vm_state", foundInstance.VMState)
	d.Set("keypair_name", foundInstance.KeypairName)

	if len(foundInstance.SecurityGroups) > 0 {
		securityGroups := make([]string, 0, len(foundInstance.SecurityGroups))
		for _, sg := range foundInstance.SecurityGroups {
			securityGroups = append(securityGroups, sg.Name)
		}
		if err := d.Set("security_groups", securityGroups); err != nil {
			return diag.FromErr(err)
		}
	}

	if len(foundInstance.Volumes) > 0 {
		volumes := make([]string, 0, len(foundInstance.Volumes))
		for _, v := range foundInstance.Volumes {
			volumes = append(volumes, v.ID)
		}
		if err := d.Set("volumes", volumes); err != nil {
			return diag.FromErr(err)
		}
	}

	if len(foundInstance.MetadataDetailed) > 0 {
		metadata := make([]map[string]interface{}, 0, len(foundInstance.MetadataDetailed))
		for _, metadataItem := range foundInstance.MetadataDetailed {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
		d.Set("metadata", metadata)
	}

	flavor := map[string]interface{}{
		"flavor_name": foundInstance.Flavor.FlavorName,
		"vcpus":       strconv.Itoa(foundInstance.Flavor.VCPUS),
		"ram":         strconv.Itoa(foundInstance.Flavor.RAM),
		"flavor_id":   foundInstance.Flavor.FlavorID,
	}
	if err := d.Set("flavor", flavor); err != nil {
		return diag.FromErr(err)
	}

	addresses := make([]map[string]string, 0, len(foundInstance.Addresses))
	for networkName, networkInfo := range foundInstance.Addresses {
		net := networkInfo[0]
		address := map[string]string{
			"network_name": networkName,
			"type":         net.Type,
			"addr":         net.Address.String(),
			"subnet_id":    net.SubnetID,
			"subnet_name":  net.SubnetName,
		}
		addresses = append(addresses, address)
	}
	if err := d.Set("addresses", addresses); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
