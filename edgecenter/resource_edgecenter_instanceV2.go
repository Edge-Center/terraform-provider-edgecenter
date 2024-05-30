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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	InstanceVolumeSizeField               = "size"
	InstanceVolumeIDField                 = "volume_id"
	InstanceBootVolumesField              = "boot_volumes"
	InstanceDataVolumesField              = "data_volumes"
	InstanceInterfaceField                = "interface"
	InstanceVMStateField                  = "vm_state"
	InstanceAddressesField                = "addresses"
	InstanceAddressesAddrField            = "addr"
	InstanceAddressesNetField             = "net"
	InstanceNameTemplateField             = "name_template"
	InstanceBootVolumesBootIndexField     = "boot_index"
	InstanceVolumesAttachmentTagField     = "attachment_tag"
	InstanceInterfaceFipSourceField       = "fip_source"
	InstanceInterfaceExistingFipIDField   = "existing_fip_id"
	InstanceInterfacePortSecDisabledField = "port_security_disabled"
	InstanceKeypairNameField              = "keypair_name"
	InstanceServerGroupField              = "server_group"
	InstanceConfigurationField            = "configuration"
	InstanceUserDataField                 = "user_data"
	InstanceAllowAppPortsField            = "allow_app_ports"
)

func resourceInstanceV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstanceCreateV2,
		ReadContext:   resourceInstanceReadV2,
		UpdateContext: resourceInstanceUpdateV2,
		DeleteContext: resourceInstanceDeleteV2,
		Description:   "A cloud instance is a virtual machine in a cloud environment.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, InstanceID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set(ProjectIDField, projectID)
				d.Set(RegionIDField, regionID)
				d.SetId(InstanceID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the instance.",
			},
			FlavorIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the flavor to be used for the instance, determining its compute and memory, for example 'g1-standard-2-4'.",
			},
			InstanceNameTemplateField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A template used to generate the instance name. This field cannot be used with 'name_templates'.",
			},
			InstanceBootVolumesField: {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						NameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name assigned to the volume. Defaults to 'system'.",
						},
						InstanceBootVolumesBootIndexField: {
							Type:         schema.TypeInt,
							Description:  "If boot_index==0 volumes can not detached. It is used only when creating an instance. This attribute can't be updated",
							Required:     true,
							ValidateFunc: validation.IntAtLeast(0),
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
							Required:    true,
							Description: "The ID of the volume.",
						},
						InstanceVolumesAttachmentTagField: {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "The block device attachment tag (exposed in the metadata).",
						},
					},
				},
			},
			InstanceDataVolumesField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem: &schema.Resource{
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
							Required:    true,
							Description: "The ID of the volume.",
						},
						InstanceVolumesAttachmentTagField: {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "The block device attachment tag (exposed in the metadata).",
						},
					},
				},
			},
			InstanceInterfaceField: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s', '%s', '%s'", edgecloudV2.InterfaceTypeSubnet, edgecloudV2.InterfaceTypeAnySubnet, edgecloudV2.InterfaceTypeExternal, edgecloudV2.InterfaceTypeReservedFixedIP),
						},
						OrderField: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Order of attaching interface",
							Computed:    true,
						},
						NetworkIDField: {
							Type:        schema.TypeString,
							Description: "Required if type is 'subnet' or 'any_subnet'.",
							Optional:    true,
							Computed:    true,
						},
						SubnetIDField: {
							Type:        schema.TypeString,
							Description: "Required if type is 'subnet'.",
							Optional:    true,
							Computed:    true,
						},
						// nested map is not supported, in this case, you do not need to use the list for the map
						InstanceInterfaceFipSourceField: {
							Type:     schema.TypeString,
							Optional: true,
						},
						InstanceInterfaceExistingFipIDField: {
							Type:     schema.TypeString,
							Optional: true,
						},
						PortIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "required if type is  'reserved_fixed_ip'",
							Optional:    true,
						},
						SecurityGroupsField: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "list of security group IDs",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						IPAddressField: {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						InstanceInterfacePortSecDisabledField: {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			InstanceKeypairNameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the key pair to be associated with the instance for SSH access.",
			},
			InstanceServerGroupField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID (uuid) of the server group to which the instance should belong.",
			},
			SecurityGroupField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of firewall configurations applied to the instance, defined by their ID and name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						IDField: {
							Type:        schema.TypeString,
							Description: "Firewall unique id (uuid)",
							Required:    true,
						},
						NameField: {
							Type:        schema.TypeString,
							Description: "Firewall name",
							Required:    true,
						},
					},
				},
			},
			PasswordField: {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{UsernameField},
				Description:  "The password to be used for accessing the instance. Required with username.",
			},
			UsernameField: {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{PasswordField},
				Description:  "The username to be used for accessing the instance. Required with password.",
			},
			MetadataMapField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			InstanceConfigurationField: {
				Type:     schema.TypeList,
				Optional: true,
				Description: `A list of key-value pairs specifying configuration settings for the instance when created 
from a template (marketplace), e.g. {"gitlab_external_url": "https://gitlab/..."}`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						KeyField: {
							Type:     schema.TypeString,
							Required: true,
						},
						ValueField: {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			InstanceUserDataField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A field for specifying user data to be used for configuring the instance at launch time.",
			},
			InstanceAllowAppPortsField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "A boolean indicating whether to allow application ports on the instance.",
			},
			FlavorField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: `A map defining the flavor of the instance, for example, {"flavor_name": "g1-standard-2-4", "ram": 4096, ...}.`,
			},
			StatusField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The current status of the instance. This is computed automatically and can be used to track the instance's state.",
			},
			InstanceVMStateField: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: fmt.Sprintf(`The current virtual machine state of the instance, 
allowing you to start or stop the VM. Possible values are %s and %s.`, InstanceVMStateStopped, InstanceVMStateActive),
				ValidateFunc: validation.StringInSlice([]string{InstanceVMStateActive, InstanceVMStateStopped}, true),
			},
			InstanceAddressesField: {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: `A list of network addresses associated with the instance, for example "pub_net": [...]`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						InstanceAddressesNetField: {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									InstanceAddressesAddrField: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The net ip address, for example '45.147.163.112'.",
									},
									TypeField: {
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
			LastUpdatedField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceInstanceCreateV2(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags = validateInstanceV2ResourceAttrs(ctx, clientV2, d)
	if diags.HasError() {
		return diags
	}

	createOpts := edgecloudV2.InstanceCreateRequest{
		Flavor:         d.Get(FlavorIDField).(string),
		KeypairName:    d.Get(InstanceKeypairNameField).(string),
		Username:       d.Get(UsernameField).(string),
		Password:       d.Get(PasswordField).(string),
		SecurityGroups: []edgecloudV2.ID{},
		ServerGroupID:  d.Get(InstanceServerGroupField).(string),
		AllowAppPorts:  d.Get(InstanceAllowAppPortsField).(bool),
	}

	if userData, ok := d.GetOk(InstanceUserDataField); ok {
		createOpts.UserData = base64.StdEncoding.EncodeToString([]byte(userData.(string)))
	}

	name := d.Get(NameField).(string)
	if len(name) > 0 {
		createOpts.Names = []string{name}
	}

	if nameTemplate, ok := d.GetOk("name_template"); ok {
		createOpts.NameTemplates = []string{nameTemplate.(string)}
	}

	bootVolumes := d.Get("boot_volumes").(*schema.Set).List()

	vs, err := extractVolumesMapV2(bootVolumes)
	if err != nil {
		return diag.FromErr(err)
	}
	createOpts.Volumes = vs

	currentDataVols := d.Get(InstanceDataVolumesField).(*schema.Set).List()
	if len(currentDataVols) > 0 {
		vs, err := extractVolumesMapV2(currentDataVols)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Volumes = append(createOpts.Volumes, vs...)
	}

	ifs := d.Get(InstanceInterfaceField).([]interface{})
	if len(ifs) > 0 {
		ifaceCreateOptsList := extractInstanceInterfaceToListCreateV2(ifs)
		createOpts.Interfaces = ifaceCreateOptsList
	}

	if metadataRaw, ok := d.GetOk(MetadataMapField); ok {
		metadata, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			diag.FromErr(err)
		}
		createOpts.Metadata = *metadata
	}

	configuration := d.Get(InstanceConfigurationField)
	if len(configuration.([]interface{})) > 0 {
		conf, err := extractKeyValueV2(configuration.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Configuration = conf
	}

	if v, ok := d.GetOk(SecurityGroupsField); ok {
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
		return diag.Errorf("error from creating instance: %s", err)
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

	resourceInstanceReadV2(ctx, d, m)

	log.Printf("[DEBUG] Finish Instance creating (%s)", instanceID)

	return diags
}

func resourceInstanceReadV2(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance reading")
	var diags diag.Diagnostics
	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set(RegionIDField, clientV2.Region)
	d.Set(ProjectIDField, clientV2.Project)

	instance, resp, err := clientV2.Instances.Get(ctx, instanceID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Removing instance %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

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
		InstanceID: instanceID,
	}
	instanceVolumes, _, err := clientV2.Volumes.List(ctx, &volumesReq)
	if err != nil {
		return diag.FromErr(err)
	}

	bootVolumesSet := d.Get(InstanceBootVolumesField).(*schema.Set)
	bootVolumesState := extractVolumesIntoMap(bootVolumesSet.List())
	enrichedBootVolumesData := EnrichVolumeData(instanceVolumes, bootVolumesState)
	if err := d.Set(InstanceBootVolumesField, schema.NewSet(bootVolumesSet.F, enrichedBootVolumesData)); err != nil {
		return diag.FromErr(err)
	}

	dataVolumesSet := d.Get(InstanceDataVolumesField).(*schema.Set)
	dataVolumesState := extractVolumesIntoMap(dataVolumesSet.List())
	enrichedDataVolumesData := EnrichVolumeData(instanceVolumes, dataVolumesState)
	if err := d.Set(InstanceDataVolumesField, schema.NewSet(dataVolumesSet.F, enrichedDataVolumesData)); err != nil {
		return diag.FromErr(err)
	}

	instancePorts, _, err := clientV2.Instances.PortsList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}
	secGroups := prepareSecurityGroupsV2(instancePorts)

	if err := d.Set(SecurityGroupField, secGroups); err != nil {
		return diag.FromErr(err)
	}

	interfacesListAPI, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	ifs := d.Get(InstanceInterfaceField).([]interface{})
	sort.Sort(instanceInterfaces(ifs))
	orderedInterfacesMap := extractInstanceInterfaceToListReadV2(ifs)
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
			i[TypeField] = interfaceOpts.InstanceInterface.Type
			i[OrderField] = interfaceOpts.Order
			i[NetworkIDField] = iFace.NetworkID
			i[SubnetIDField] = subnetID
			i[PortIDField] = iFace.PortID
			i[InstanceInterfacePortSecDisabledField] = !iFace.PortSecurityEnabled

			if interfaceOpts.InstanceInterface.FloatingIP != nil {
				i[InstanceInterfaceFipSourceField] = interfaceOpts.InstanceInterface.FloatingIP.Source
				i[InstanceInterfaceExistingFipIDField] = interfaceOpts.InstanceInterface.FloatingIP.ExistingFloatingID
			}
			i[IPAddressField] = assignment.IPAddress.String()
			if port, err := findInstancePortV2(portID, instancePorts); err == nil {
				sgs := make([]string, len(port.SecurityGroups))
				for i, sg := range port.SecurityGroups {
					sgs[i] = sg.ID
				}
				i[SecurityGroupsField] = sgs
			}

			interfacesList = append(interfacesList, i)
		}
	}
	if err := d.Set(InstanceInterfaceField, interfacesList); err != nil {
		return diag.FromErr(err)
	}

	metadata := d.Get(MetadataMapField).(map[string]interface{})
	newMetadata := make(map[string]interface{}, len(metadata))
	for k := range metadata {
		md, _, err := clientV2.Instances.MetadataGetItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: k})
		if err != nil {
			return diag.Errorf("cannot get metadata with key: %s. Error: %s", instanceID, err)
		}
		newMetadata[k] = md.Value
	}
	if err := d.Set(MetadataMapField, newMetadata); err != nil {
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

func resourceInstanceUpdateV2(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance updating")
	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validateInstanceV2ResourceAttrs(ctx, clientV2, d)
	if diags.HasError() {
		return diags
	}

	if d.HasChange(NameField) {
		nameTemplate := d.Get(InstanceNameTemplateField).(string)
		if len(nameTemplate) == 0 {
			opts := edgecloudV2.Name{Name: d.Get(NameField).(string)}
			if _, _, err := clientV2.Instances.Rename(ctx, instanceID, &opts); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(FlavorIDField) {
		flavorID := d.Get(FlavorIDField).(string)
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

	if d.HasChange(MetadataMapField) {
		omd, nmd := d.GetChange(MetadataMapField)
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

	if d.HasChange(InstanceInterfaceField) {
		iOldRaw, iNewRaw := d.GetChange(InstanceInterfaceField)
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

				sgsIDsOld := getSecurityGroupsIDsV2(iOld[SecurityGroupsField].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew[SecurityGroupsField].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld[PortIDField].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}
					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{SecurityGroupsField, InstanceInterfacePortSecDisabledField})
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

				sgsIDsOld := getSecurityGroupsIDsV2(iOld[SecurityGroupsField].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew[SecurityGroupsField].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld[PortIDField].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}

					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{SecurityGroupsField, InstanceInterfacePortSecDisabledField})
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

				sgsIDsOld := getSecurityGroupsIDsV2(iOld[SecurityGroupsField].([]interface{}))
				sgsIDsNew := getSecurityGroupsIDsV2(iNew[SecurityGroupsField].([]interface{}))
				if len(sgsIDsOld) > 0 || len(sgsIDsNew) > 0 {
					portID := iOld[PortIDField].(string)
					removeSGs := getSecurityGroupsDifferenceV2(sgsIDsNew, sgsIDsOld)
					if err := removeSecurityGroupFromInstanceV2(ctx, clientV2, instanceID, portID, removeSGs); err != nil {
						return diag.FromErr(err)
					}

					addSGs := getSecurityGroupsDifferenceV2(sgsIDsOld, sgsIDsNew)
					if err := attachSecurityGroupToInstanceV2(ctx, clientV2, instanceID, portID, addSGs); err != nil {
						return diag.FromErr(err)
					}
				}

				differentFields := getMapDifference(iOld, iNew, []string{SecurityGroupsField, InstanceInterfacePortSecDisabledField})
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

	if d.HasChange(InstanceServerGroupField) {
		oldSGRaw, newSGRaw := d.GetChange(InstanceServerGroupField)
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

	if d.HasChange(InstanceBootVolumesField) {
		oldVolumesRaw, newVolumesRaw := d.GetChange(InstanceBootVolumesField)
		err = UpdateVolumes(ctx, d, clientV2, instanceID, oldVolumesRaw, newVolumesRaw)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(InstanceDataVolumesField) {
		oldVolumesRaw, newVolumesRaw := d.GetChange(InstanceDataVolumesField)
		err = UpdateVolumes(ctx, d, clientV2, instanceID, oldVolumesRaw, newVolumesRaw)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(InstanceVMStateField) {
		state := d.Get(InstanceVMStateField).(string)
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

	d.Set(LastUpdatedField, time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish Instance updating")

	return resourceInstanceReadV2(ctx, d, m)
}

func resourceInstanceDeleteV2(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Instance deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
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
