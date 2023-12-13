package instance

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func instanceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the project",
		},
		"region_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the region",
		},
		"name": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"name_templates"},
			Description:   "the instance name",
		},
		"name_templates": {
			Type:          schema.TypeList,
			Optional:      true,
			ConflictsWith: []string{"name"},
			Description:   "list of the instance names which will be changed by template: ip_octets, two_ip_octets, one_ip_octet",
			Elem:          &schema.Schema{Type: schema.TypeString},
		},
		"flavor": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "ID of the flavor, determining its compute and memory, for example 'g1-standard-2-4'.",
		},
		"interface": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "list defining the network interfaces to be attached to the instance",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Type:     schema.TypeString,
						Required: true,
						Description: fmt.Sprintf(
							"available values are '%s', '%s', '%s', '%s'",
							edgecloud.InterfaceTypeSubnet, edgecloud.InterfaceTypeAnySubnet,
							edgecloud.InterfaceTypeExternal, edgecloud.InterfaceTypeReservedFixedIP,
						),
					},
					"security_groups": {
						Type:        schema.TypeList,
						Optional:    true,
						Computed:    true,
						Description: "list of security group IDs",
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
					"network_id": {
						Type:         schema.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.IsUUID,
						Description: fmt.Sprintf(
							"ID of the network that the subnet belongs to, required if type is '%s' or '%s'",
							edgecloud.InterfaceTypeSubnet,
							edgecloud.InterfaceTypeAnySubnet,
						),
					},
					"subnet_id": {
						Type:         schema.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.IsUUID,
						Description:  fmt.Sprintf("required if type is '%s'", edgecloud.InterfaceTypeSubnet),
					},
					"floating_ip_source": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "floating IP type: 'existing' or 'new'",
					},
					"floating_ip": {
						Type:         schema.TypeString,
						ValidateFunc: validation.IsUUID,
						Optional:     true,
						Computed:     true,
						Description:  "floating IP for this subnet attachment",
					},
					"port_id": {
						Type:         schema.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validation.IsUUID,
						Description:  fmt.Sprintf("required if type is '%s'", edgecloud.InterfaceTypeReservedFixedIP),
					},
					"ip_address": {
						Type:     schema.TypeString,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
		"volume": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "list of volumes for the instances",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"source": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "volume source",
						ValidateFunc: validation.StringInSlice(
							[]string{"new-volume", "existing-volume", "image"}, false,
						),
					},
					"size": {
						Type:         schema.TypeInt,
						Required:     true,
						Description:  "size of the volume, specified in gigabytes (GB)",
						ValidateFunc: validation.IntAtLeast(1),
					},
					"name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "name of the volume",
					},
					"type_name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "volume type with valid values. defaults to 'ssd_hiiops'",
						ValidateFunc: validation.StringInSlice([]string{
							string(edgecloud.VolumeTypeSsdHiIops),
							string(edgecloud.VolumeTypeSsdLocal),
							string(edgecloud.VolumeTypeUltra),
							string(edgecloud.VolumeTypeCold),
							string(edgecloud.VolumeTypeStandard),
						}, false),
						Default: string(edgecloud.VolumeTypeSsdHiIops),
					},
					"attachment_tag": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "the block device attachment tag (exposed in the metadata)",
					},
					"boot_index": {
						Type: schema.TypeInt,
						Description: `0 for the primary boot device. 
unique positive values for other bootable devices. negative - the boot is prohibited`,
						Optional: true,
					},
					"image_id": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.IsUUID,
						Description:  "ID of the image. this field is mandatory if creating a volume from an image",
					},
					"volume_id": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.IsUUID,
						Description:  "ID of the volume. this field is mandatory if the volume is a pre-existing volume",
					},
					"metadata": {
						Type:        schema.TypeMap,
						Optional:    true,
						Description: "map containing metadata, for example tags.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					// computed attributes
					"id": {
						Type:     schema.TypeString,
						Optional: true,
						Computed: true,
					},
					"delete_on_termination": {
						Type:     schema.TypeBool,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
		"metadata": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "map containing metadata, for example tags.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"keypair_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "the name of the keypair to inject into new instance(s)",
		},
		"server_group_id": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.IsUUID,
			Description:  "UUID of the anti-affinity or affinity server group (placement groups)",
		},
		"security_groups": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "list of security group (firewall) UUIDs",
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.IsUUID,
			},
		},
		"user_data": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "a string in the base64 format. examples of user_data: https://cloudinit.readthedocs.io/en/latest/topics/examples.html",
		},
		"username": {
			Type:         schema.TypeString,
			Optional:     true,
			RequiredWith: []string{"password"},
			Description:  "name of a new user on a Linux VM",
		},
		"password": {
			Type:         schema.TypeString,
			Optional:     true,
			RequiredWith: []string{"username"},
			Description: `this parameter is used to set the password either for the 'Admin' user on a Windows VM or
the default user or a new user on a Linux VM`,
		},
		"allow_app_ports": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "if true, application ports will be allowed in the security group for the instances created from the marketplace application template",
		},
		// computed attributes
		"region": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the region",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "status of the VM",
		},
		"vm_state": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "state of the virtual machine",
		},
		"addresses": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "network addresses associated with the instance",
			Elem:        &schema.Schema{Type: schema.TypeMap},
		},
		"keypair_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "uuid of the keypair",
		},
		"metadata_detailed": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "metadata in detailed format with system info",
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
	}
}
