package edgecenter

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	InstanceCreateTimeout = 10 * time.Minute
	InstanceDeleteTimeout = 10 * time.Minute
	InstanceUpdateTimeout = 10 * time.Minute
	InstancePoint         = "instances"

	InstanceVMStateActive  = "active"
	InstanceVMStateStopped = "stopped"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext:      resourceInstanceCreate,
		ReadContext:        resourceInstanceRead,
		UpdateContext:      resourceInstanceUpdate,
		DeleteContext:      resourceInstanceDelete,
		Description:        "**WARNING:** Resource \"instance\" is deprecated and unavailable.\n Use edgecenter_instanceV2 resource instead.\n\n A cloud instance is a virtual machine in a cloud environment.",
		DeprecationMessage: "!> **WARNING:** This resource is deprecated and will be removed in the next major version. Use edgecenter_instanceV2 resource instead",

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, InstanceID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(InstanceID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the instance.",
			},
			"flavor_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the flavor to be used for the instance, determining its compute and memory, for example 'g1-standard-2-4'.",
			},
			"name_templates": {
				Type:          schema.TypeList,
				Optional:      true,
				Deprecated:    "Use name_template instead.",
				ConflictsWith: []string{"name_template"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"name_template": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_templates"},
				Description:   "A template used to generate the instance name. This field cannot be used with 'name_templates'.",
			},
			"volume": {
				Type:        schema.TypeSet,
				Required:    true,
				Set:         volumeUniqueID,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The name assigned to the volume. Defaults to 'system'.",
						},
						"source": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Currently available only 'existing-volume' value",
							ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
								v := val.(string)
								if edgecloudV2.VolumeSource(v) == edgecloudV2.VolumeSourceExistingVolume {
									return diag.Diagnostics{}
								}
								return diag.Errorf("wrong source type %s, now available values is '%s'", v, edgecloudV2.VolumeSourceExistingVolume)
							},
						},
						"boot_index": {
							Type:        schema.TypeInt,
							Description: "If boot_index==0 volumes can not detached",
							Optional:    true,
						},
						"type_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The type of volume to create. Valid values are 'ssd_hiiops', 'standard', 'cold', and 'ultra'. Defaults to 'standard'.",
						},
						"image_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "The size of the volume, specified in gigabytes (GB).",
						},
						"volume_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"attachment_tag": {
							Type:     schema.TypeString,
							Optional: true,
						},
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
			"interface": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s', '%s', '%s'", edgecloudV2.InterfaceTypeSubnet, edgecloudV2.InterfaceTypeAnySubnet, edgecloudV2.InterfaceTypeExternal, edgecloudV2.InterfaceTypeReservedFixedIP),
						},
						"order": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Order of attaching interface",
							Computed:    true,
						},
						"network_id": {
							Type:        schema.TypeString,
							Description: "Required if type is 'subnet' or 'any_subnet'.",
							Optional:    true,
							Computed:    true,
						},
						"subnet_id": {
							Type:        schema.TypeString,
							Description: "Required if type is 'subnet'.",
							Optional:    true,
							Computed:    true,
						},
						// nested map is not supported, in this case, you do not need to use the list for the map
						"fip_source": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"existing_fip_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"port_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "required if type is  'reserved_fixed_ip'",
							Optional:    true,
						},
						"security_groups": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "list of security group IDs",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						"port_security_disabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"keypair_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the key pair to be associated with the instance for SSH access.",
			},
			"server_group": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID (uuid) of the server group to which the instance should belong.",
			},
			"security_group": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of firewall configurations applied to the instance, defined by their ID and name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "Firewall unique id (uuid)",
							Required:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Firewall name",
							Required:    true,
						},
					},
				},
			},
			"password": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"username"},
				Description:  "The password to be used for accessing the instance. Required with username.",
			},
			"username": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"password"},
				Description:  "The username to be used for accessing the instance. Required with password.",
			},
			"metadata": {
				Type:          schema.TypeList,
				Optional:      true,
				Deprecated:    "Use metadata_map instead",
				ConflictsWith: []string{"metadata_map"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"metadata_map": {
				Type:          schema.TypeMap,
				Optional:      true,
				ConflictsWith: []string{"metadata"},
				Description:   "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Description: `A list of key-value pairs specifying configuration settings for the instance when created 
from a template (marketplace), e.g. {"gitlab_external_url": "https://gitlab/..."}`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"userdata": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "**Deprecated**",
				Deprecated:    "Use user_data instead",
				ConflictsWith: []string{"user_data"},
			},
			"user_data": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"userdata"},
				Description:   "A field for specifying user data to be used for configuring the instance at launch time.",
			},
			"allow_app_ports": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "A boolean indicating whether to allow application ports on the instance.",
			},
			"flavor": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: `A map defining the flavor of the instance, for example, {"flavor_name": "g1-standard-2-4", "ram": 4096, ...}.`,
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The current status of the instance. This is computed automatically and can be used to track the instance's state.",
			},
			"vm_state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: fmt.Sprintf(`The current virtual machine state of the instance, 
allowing you to start or stop the VM. Possible values are %s and %s.`, InstanceVMStateStopped, InstanceVMStateActive),
				ValidateFunc: validation.StringInSlice([]string{InstanceVMStateActive, InstanceVMStateStopped}, true),
			},
			"addresses": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: `A list of network addresses associated with the instance, for example "pub_net": [...]`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"net": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"addr": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The net ip address, for example '45.147.163.112'.",
									},
									"type": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The net type, for example 'fixed'.",
									},
								},
							},
						},
					},
				},
			},
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceInstanceCreate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_instance\" is deprecated and unavailable. Use edgecenter_instanceV2 resource instead"))
}

func resourceInstanceRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_instance\" is deprecated and unavailable. Use edgecenter_instanceV2 resource instead"))
}

func resourceInstanceUpdate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_instance\" is deprecated and unavailable. Use edgecenter_instanceV2 resource instead"))
}

func resourceInstanceDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("resource \"edgecenter_instance\" is deprecated and unavailable. Use edgecenter_instanceV2 resource instead"))
}
