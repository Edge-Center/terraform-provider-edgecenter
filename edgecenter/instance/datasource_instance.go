package instance

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
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
			"metadata_detailed": {
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
			"server_group_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "UUID of the anti-affinity or affinity server group (placement groups)",
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
	d.Set("status", foundInstance.Status)
	d.Set("region", foundInstance.Region)
	d.Set("vm_state", foundInstance.VMState)
	d.Set("keypair_name", foundInstance.KeypairName)

	if err := setSecurityGroups(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
	}

	if err := setMetadataDetailed(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
	}

	if err := setFlavor(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
	}

	if err := setAddresses(ctx, d, foundInstance); err != nil {
		return diag.FromErr(err)
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
	if err = d.Set("interface", cleanInterfaces); err != nil {
		return diag.FromErr(err)
	}

	sg, err := util.ServerGroupGetByInstance(ctx, client, d.Id())
	if err != nil {
		if !errors.Is(err, util.ErrServerGroupNotFound) {
			return diag.Errorf("Error retrieving instance server groups: %s", err)
		}
	}

	if sg != nil {
		d.Set("server_group_id", sg.ID)
	}

	return nil
}
