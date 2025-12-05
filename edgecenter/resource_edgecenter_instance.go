package edgecenter

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	InstanceCreateTimeout  = 10 * time.Minute
	InstanceDeleteTimeout  = 10 * time.Minute
	InstanceUpdateTimeout  = 10 * time.Minute
	InstanceMigrateTimeout = 10 * time.Minute
	InstancePoint          = "instances"

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

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	diags = validateInstanceResourceAttrs(d)
	if diags.HasError() {
		return diags
	}

	createOpts := edgecloudV2.InstanceCreateRequest{
		Flavor:         d.Get("flavor_id").(string),
		KeypairName:    d.Get("keypair_name").(string),
		Username:       d.Get("username").(string),
		Password:       d.Get("password").(string),
		SecurityGroups: []edgecloudV2.ID{},
		ServerGroupID:  d.Get("server_group").(string),
		AllowAppPorts:  d.Get("allow_app_ports").(bool),
	}

	if userData, ok := d.GetOk("user_data"); ok {
		createOpts.UserData = base64.StdEncoding.EncodeToString([]byte(userData.(string)))
	} else if userData, ok := d.GetOk("userdata"); ok {
		createOpts.UserData = base64.StdEncoding.EncodeToString([]byte(userData.(string)))
	}

	name := d.Get("name").(string)
	if len(name) > 0 {
		createOpts.Names = []string{name}
	}

	if nameTemplatesRaw, ok := d.GetOk("name_templates"); ok {
		nameTemplates := nameTemplatesRaw.([]interface{})
		if len(nameTemplates) > 0 {
			NameTemp := make([]string, len(nameTemplates))
			for i, nametemp := range nameTemplates {
				NameTemp[i] = nametemp.(string)
			}
			createOpts.NameTemplates = NameTemp
		}
	} else if nameTemplate, ok := d.GetOk("name_template"); ok {
		createOpts.NameTemplates = []string{nameTemplate.(string)}
	}

	currentVols := d.Get("volume").(*schema.Set).List()
	if len(currentVols) > 0 {
		vs, err := extractVolumesMapV2(currentVols)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Volumes = vs
	}

	ifs := d.Get("interface").([]interface{})
	if len(ifs) > 0 {
		ifaceCreateOptsList := extractInstanceInterfaceToListCreate(ifs)
		createOpts.Interfaces = ifaceCreateOptsList
	}

	if v, ok := d.GetOk("metadata"); ok {
		metadataKV, err := extractKeyValueV2(v.([]interface{}))
		if err != nil {
			diag.FromErr(err)
		}
		metadata, err := MapInterfaceToMapString(metadataKV)
		if err != nil {
			diag.FromErr(err)
		}
		createOpts.Metadata = *metadata
	} else if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		metadata, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			diag.FromErr(err)
		}
		createOpts.Metadata = *metadata
	}

	configuration := d.Get("configuration")
	if len(configuration.([]interface{})) > 0 {
		conf, err := extractKeyValueV2(configuration.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Configuration = conf
	}

	if v, ok := d.GetOk("security_groups"); ok {
		securityGroups := v.([]interface{})
		sgsList := make([]edgecloudV2.ID, 0, len(securityGroups))
		for _, sg := range securityGroups {
			sgsList = append(sgsList, edgecloudV2.ID{ID: sg.(string)})
		}
		createOpts.SecurityGroups = sgsList
	}

	log.Printf("[DEBUG] Instance create options: %+v", createOpts)

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Instances.Create, &createOpts, clientV2, InstanceCreateTimeout)
	if err != nil {
		return diag.Errorf("error creating instance: %s", err)
	}

	instanceID := taskResult.Instances[0]
	log.Printf("[DEBUG] Instance id (%s)", instanceID)
	d.SetId(instanceID)

	// Code below adjusts all interfaces PortSecurityDisabled opt
	log.Println("[DEBUG] ports security options adjusting...")
	diagsAdjust := adjustAllPortsSecurityDisabledOpt(ctx, clientV2, instanceID, ifs)
	if len(diagsAdjust) != 0 {
		return append(diags, diagsAdjust...)
	}

	resourceInstanceRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Instance creating (%s)", instanceID)

	return diags
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance reading")
	var diags diag.Diagnostics

	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("region_id", clientV2.Region)
	d.Set("project_id", clientV2.Project)

	instance, resp, err := clientV2.Instances.Get(ctx, instanceID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Removing instance %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", instance.Name)
	d.Set("flavor_id", instance.Flavor.FlavorID)
	d.Set("status", instance.Status)
	d.Set("vm_state", instance.VMState)

	flavor := make(map[string]interface{}, 4)
	flavor["flavor_id"] = instance.Flavor.FlavorID
	flavor["flavor_name"] = instance.Flavor.FlavorName
	flavor["ram"] = strconv.Itoa(instance.Flavor.RAM)
	flavor["vcpus"] = strconv.Itoa(instance.Flavor.VCPUS)
	d.Set("flavor", flavor)

	currentVolumes := extractVolumesIntoMap(d.Get("volume").(*schema.Set).List())

	extVolumes := make([]interface{}, 0, len(instance.Volumes))
	for _, vol := range instance.Volumes {
		v, ok := currentVolumes[vol.ID]
		// todo fix it
		if !ok {
			v = make(map[string]interface{})
			v["volume_id"] = vol.ID
			v["source"] = edgecloudV2.VolumeSourceExistingVolume
		}

		v["id"] = vol.ID
		v["delete_on_termination"] = vol.DeleteOnTermination
		extVolumes = append(extVolumes, v)
	}

	if err := d.Set("volume", schema.NewSet(volumeUniqueID, extVolumes)); err != nil {
		return diag.FromErr(err)
	}

	instancePorts, _, err := clientV2.Instances.PortsList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}
	secGroups := prepareSecurityGroupsV2(instancePorts)

	if err := d.Set("security_group", secGroups); err != nil {
		return diag.FromErr(err)
	}

	interfacesListAPI, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	ifs := d.Get("interface").([]interface{})
	sort.Sort(instanceInterfaces(ifs))
	orderedInterfacesMap := extractInstanceInterfaceToListRead(ifs)
	var interfacesList []interface{}
	for _, iFace := range interfacesListAPI {
		if len(iFace.IPAssignments) == 0 {
			continue
		}

		portID := iFace.PortID
		for _, assignment := range iFace.IPAssignments {
			subnetID := assignment.SubnetID

			var interfaceOpts OrderedInterfaceOpts
			var orderedInterfaceOpts OrderedInterfaceOpts
			var ok bool

			// we need to match our interfaces with api's interfaces
			// but with don't have any unique value, that's why we use exactly that list of keys
			for _, k := range []string{subnetID, iFace.PortID, iFace.NetworkID, string(edgecloudV2.InterfaceTypeExternal)} {
				if orderedInterfaceOpts, ok = orderedInterfacesMap[k]; ok {
					interfaceOpts = orderedInterfaceOpts
					break
				}
			}

			if !ok {
				continue
			}

			i := make(map[string]interface{})
			i["type"] = interfaceOpts.InstanceInterface.Type
			i["order"] = interfaceOpts.Order
			i["network_id"] = iFace.NetworkID
			i["subnet_id"] = subnetID
			i["port_id"] = iFace.PortID
			i["port_security_disabled"] = !iFace.PortSecurityEnabled

			if interfaceOpts.InstanceInterface.FloatingIP != nil {
				i["fip_source"] = interfaceOpts.InstanceInterface.FloatingIP.Source
				i["existing_fip_id"] = interfaceOpts.InstanceInterface.FloatingIP.ExistingFloatingID
			}
			i["ip_address"] = assignment.IPAddress.String()
			if port, err := findInstancePortV2(portID, instancePorts); err == nil {
				sgs := make([]string, len(port.SecurityGroups))
				for i, sg := range port.SecurityGroups {
					sgs[i] = sg.ID
				}
				i["security_groups"] = sgs
			}

			interfacesList = append(interfacesList, i)
		}
	}
	if err := d.Set("interface", interfacesList); err != nil {
		return diag.FromErr(err)
	}

	if metadataRaw, ok := d.GetOk("metadata"); ok {
		metadata := metadataRaw.([]interface{})
		sliced := make([]map[string]string, len(metadata))
		for i, data := range metadata {
			d := data.(map[string]interface{})
			mdata := make(map[string]string, 2)

			md, _, err := clientV2.Instances.MetadataGetItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: d["key"].(string)})
			if err != nil {
				return diag.Errorf("cannot get metadata with key: %s. Error: %s", d["key"].(string), err)
			}
			mdata["key"] = md.Key
			mdata["value"] = md.Value
			sliced[i] = mdata
		}
		d.Set("metadata", sliced)
	} else {
		metadata := d.Get("metadata_map").(map[string]interface{})
		newMetadata := make(map[string]interface{}, len(metadata))
		for k := range metadata {
			md, _, err := clientV2.Instances.MetadataGetItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: k})
			if err != nil {
				return diag.Errorf("cannot get metadata with key: %s. Error: %s", instanceID, err)
			}
			newMetadata[k] = md.Value
		}
		if err := d.Set("metadata_map", newMetadata); err != nil {
			return diag.FromErr(err)
		}
	}

	addresses := []map[string][]map[string]string{}
	for _, data := range instance.Addresses {
		d := map[string][]map[string]string{}
		netd := make([]map[string]string, len(data))
		for i, iaddr := range data {
			ndata := make(map[string]string, 2)
			ndata["type"] = iaddr.Type
			ndata["addr"] = iaddr.Address.String()
			netd[i] = ndata
		}
		d["net"] = netd
		addresses = append(addresses, d)
	}
	if err := d.Set("addresses", addresses); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Instance reading")

	return diags
}

func resourceInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance updating")
	instanceID := d.Id()

	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validateInstanceResourceAttrs(d)
	if diags.HasError() {
		return diags
	}

	if d.HasChange("name") {
		nameTemplates := d.Get("name_templates").([]interface{})
		nameTemplate := d.Get("name_template").(string)
		if len(nameTemplate) == 0 && len(nameTemplates) == 0 {
			opts := edgecloudV2.Name{Name: d.Get("name").(string)}
			if _, _, err := clientV2.Instances.Rename(ctx, instanceID, &opts); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("flavor_id") {
		flavorID := d.Get("flavor_id").(string)
		result, _, err := clientV2.Instances.UpdateFlavor(ctx, instanceID, &edgecloudV2.InstanceFlavorUpdateRequest{FlavorID: flavorID})
		if err != nil {
			return diag.FromErr(err)
		}
		taskID := result.Tasks[0]
		log.Printf("[DEBUG] Task id (%s)", taskID)
		task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, InstanceUpdateTimeout)
		if err != nil {
			return diag.FromErr(err)
		}

		if task.State == edgecloudV2.TaskStateError {
			return diag.Errorf("cannot update flavor in instance with ID: %s", instanceID)
		}
	}

	if d.HasChange("metadata") {
		omd, nmd := d.GetChange("metadata")
		if len(omd.([]interface{})) > 0 {
			for _, data := range omd.([]interface{}) {
				d := data.(map[string]interface{})
				k := d["key"].(string)
				_, err = clientV2.Instances.MetadataDeleteItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: k})
				if err != nil {
					return diag.Errorf("cannot delete metadata key: %s. Error: %s", k, err)
				}
			}
		}
		MetaData := make(edgecloudV2.Metadata)
		if len(nmd.([]interface{})) > 0 {
			for _, data := range nmd.([]interface{}) {
				d := data.(map[string]interface{})
				MetaData[d["key"].(string)] = d["value"].(string)
			}

			_, err = clientV2.Instances.MetadataCreate(ctx, instanceID, &MetaData)
			if err != nil {
				return diag.Errorf("cannot create metadata. Error: %s", err)
			}
		}
	} else if d.HasChange("metadata_map") {
		omd, nmd := d.GetChange("metadata_map")
		if !reflect.DeepEqual(omd, nmd) {
			MetaData := make(edgecloudV2.Metadata)
			for k, v := range nmd.(map[string]interface{}) {
				MetaData[k] = v.(string)
			}
			_, err = clientV2.Instances.MetadataCreate(ctx, instanceID, &MetaData)
			if err != nil {
				return diag.Errorf("cannot create metadata. Error: %s", err)
			}
		}
	}

	if d.HasChange("interface") {
		iOldRaw, iNewRaw := d.GetChange("interface")
		ifsOldSlice, ifsNewSlice := iOldRaw.([]interface{}), iNewRaw.([]interface{})
		sort.Sort(instanceInterfaces(ifsOldSlice))
		sort.Sort(instanceInterfaces(ifsNewSlice))

		diagsAdjust := adjustAllPortsSecurityDisabledOpt(ctx, clientV2, instanceID, ifsNewSlice)
		if len(diagsAdjust) != 0 {
			return diagsAdjust
		}

		switch {
		// the same number of interfaces
		case len(ifsOldSlice) == len(ifsNewSlice):
			for idx, item := range ifsOldSlice {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDsV2(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}
					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{"security_groups", "port_security_disabled"})
				if len(differentFields) > 0 {
					if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iOld); err != nil {
						return diag.FromErr(err)
					}
					if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

		// new interfaces > old interfaces - need to attach new
		case len(ifsOldSlice) < len(ifsNewSlice):
			for idx, item := range ifsOldSlice {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDsV2(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}

					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{"security_groups", "port_security_disabled"})
				if len(differentFields) > 0 {
					if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iOld); err != nil {
						return diag.FromErr(err)
					}
					if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

			for _, item := range ifsNewSlice[len(ifsOldSlice):] {
				iNew := item.(map[string]interface{})
				if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iNew); err != nil {
					return diag.FromErr(err)
				}
			}

		// old interfaces > new interfaces - need to detach old
		case len(ifsOldSlice) > len(ifsNewSlice):
			for idx, item := range ifsOldSlice[:len(ifsNewSlice)] {
				iOld := item.(map[string]interface{})
				iNew := ifsNewSlice[idx].(map[string]interface{})

				sgsIDsOld := getSecurityGroupsIDsV2(iOld["security_groups"].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew["security_groups"].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld["port_id"].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}

					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{"security_groups", "port_security_disabled"})
				if len(differentFields) > 0 {
					if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iOld); err != nil {
						return diag.FromErr(err)
					}
					if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iNew); err != nil {
						return diag.FromErr(err)
					}
				}
			}

			for _, item := range ifsOldSlice[len(ifsNewSlice):] {
				iOld := item.(map[string]interface{})
				if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iOld); err != nil {
					return diag.FromErr(err)
				}
			}
		}
		diagsAdjust = adjustAllPortsSecurityDisabledOpt(ctx, clientV2, instanceID, ifsNewSlice)
		if len(diagsAdjust) != 0 {
			return diagsAdjust
		}
	}

	if d.HasChange("server_group") {
		oldSGRaw, newSGRaw := d.GetChange("server_group")
		oldSGID, newSGID := oldSGRaw.(string), newSGRaw.(string)

		// delete old server group
		if oldSGID != "" {
			err := deleteServerGroupV2(ctx, clientV2, instanceID)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		// add new server group if needed
		if newSGID != "" {
			err := addServerGroupV2(ctx, clientV2, instanceID, newSGID)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("volume") {
		oldVolumesRaw, newVolumesRaw := d.GetChange("volume")
		oldVolumes := extractInstanceVolumesMap(oldVolumesRaw.(*schema.Set).List())
		newVolumes := extractInstanceVolumesMap(newVolumesRaw.(*schema.Set).List())

		vDetachOpts := edgecloudV2.VolumeDetachRequest{InstanceID: d.Id()}
		for vid := range oldVolumes {
			if isAttached := newVolumes[vid]; isAttached {
				// mark as already attached
				newVolumes[vid] = false
				continue
			}
			if _, _, err := clientV2.Volumes.Detach(ctx, vid, &vDetachOpts); err != nil {
				return diag.FromErr(err)
			}
		}

		// range over not attached volumes
		vAttachOpts := edgecloudV2.VolumeAttachRequest{InstanceID: d.Id()}
		for vid, ok := range newVolumes {
			if ok {
				if _, _, err := clientV2.Volumes.Attach(ctx, vid, &vAttachOpts); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChange("vm_state") {
		state := d.Get("vm_state").(string)
		switch state {
		case InstanceVMStateActive:
			if _, _, err := clientV2.Instances.InstanceStart(ctx, instanceID); err != nil {
				return diag.FromErr(err)
			}
			startStateConf := &retry.StateChangeConf{
				Target:     []string{InstanceVMStateActive},
				Refresh:    ServerV2StateRefreshFuncV2(ctx, clientV2, instanceID),
				Timeout:    d.Timeout(schema.TimeoutCreate),
				Delay:      10 * time.Second,
				MinTimeout: 3 * time.Second,
			}
			_, err = startStateConf.WaitForStateContext(ctx)
			if err != nil {
				return diag.Errorf("Error waiting for instance (%s) to become active: %s", d.Id(), err)
			}
		case InstanceVMStateStopped:
			if _, _, err := clientV2.Instances.InstanceStop(ctx, instanceID); err != nil {
				return diag.FromErr(err)
			}
			stopStateConf := &retry.StateChangeConf{
				Target:     []string{InstanceVMStateStopped},
				Refresh:    ServerV2StateRefreshFuncV2(ctx, clientV2, instanceID),
				Timeout:    d.Timeout(schema.TimeoutCreate),
				Delay:      10 * time.Second,
				MinTimeout: 3 * time.Second,
			}
			_, err = stopStateConf.WaitForStateContext(ctx)
			if err != nil {
				return diag.Errorf("Error waiting for instance (%s) to become inactive(stopped): %s", d.Id(), err)
			}
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish Instance updating")

	return resourceInstanceRead(ctx, d, m)
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	var delOpts edgecloudV2.InstanceDeleteOptions
	results, _, err := clientV2.Instances.Delete(ctx, instanceID, &delOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, InstanceDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete instance with ID: %s", instanceID)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of Instance deleting")

	return diags
}
