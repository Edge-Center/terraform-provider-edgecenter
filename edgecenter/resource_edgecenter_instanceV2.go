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
	InstanceVolumeSizeField           = "size"
	InstanceVolumeIDField             = "volume_id"
	InstanceBootVolumesField          = "boot_volumes"
	InstanceDataVolumesField          = "data_volumes"
	InstanceInterfacesField           = "interfaces"
	InstanceVMStateField              = "vm_state"
	InstanceAddressesField            = "addresses"
	InstanceAddressesAddrField        = "addr"
	InstanceAddressesNetField         = "net"
	InstanceNameTemplateField         = "name_template"
	InstanceBootVolumesBootIndexField = "boot_index"
	InstanceVolumesAttachmentTagField = "attachment_tag"
	InstanceInterfaceFipSourceField   = "fip_source"
	InstanceKeypairNameField          = "keypair_name"
	InstanceServerGroupField          = "server_group"
	InstanceConfigurationField        = "configuration"
	InstanceUserDataField             = "user_data"
	InstanceAllowAppPortsField        = "allow_app_ports"
	InstanceReservedFixedIPPortID     = "reserved_fixed_ip_port_id"
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
			InstanceInterfacesField: {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  fmt.Sprintf("Available values are '%s', '%s', '%s', '%s'", edgecloudV2.InterfaceTypeSubnet, edgecloudV2.InterfaceTypeAnySubnet, edgecloudV2.InterfaceTypeExternal, edgecloudV2.InterfaceTypeReservedFixedIP),
							ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.InterfaceTypeSubnet), string(edgecloudV2.InterfaceTypeAnySubnet), string(edgecloudV2.InterfaceTypeExternal), string(edgecloudV2.InterfaceTypeReservedFixedIP)}, true),
						},
						OrderField: {
							Computed:    true,
							Type:        schema.TypeInt,
							Description: "Order of attaching interface.",
						},
						IsDefaultField: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "This field determines whether this interface will be connected first. The first connected interface defines the default routing. If you change this attribute, the IP address of interfaces connected earlier than the selected new default interface will change, if the reserved IP address is not used in these interfaces.",
						},
						NetworkIDField: {
							Type:         schema.TypeString,
							Description:  "Required if type is 'subnet' or 'any_subnet'.",
							Optional:     true,
							Default:      "",
							ValidateFunc: validation.IsUUID,
						},
						NetworkNameField: {
							Type:        schema.TypeString,
							Description: "Name of the network.",
							Computed:    true,
						},
						SubnetIDField: {
							Type:         schema.TypeString,
							Description:  "Required if type is 'subnet'.",
							Optional:     true,
							Default:      "",
							ValidateFunc: validation.IsUUID,
						},
						InstanceReservedFixedIPPortID: {
							Default:      "",
							Type:         schema.TypeString,
							Description:  "required if type is  'reserved_fixed_ip'",
							Optional:     true,
							ValidateFunc: validation.IsUUID,
						},
						IPAddressField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP address of the interface.",
						},
						PortIDField: {
							Type:     schema.TypeString,
							Computed: true,
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
		Flavor:      d.Get(FlavorIDField).(string),
		KeypairName: d.Get(InstanceKeypairNameField).(string),
		Username:    d.Get(UsernameField).(string),
		Password:    d.Get(PasswordField).(string),
		// SecurityGroups: []edgecloudV2.ID{},
		ServerGroupID: d.Get(InstanceServerGroupField).(string),
		AllowAppPorts: d.Get(InstanceAllowAppPortsField).(bool),
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

	ifsRaw := d.Get(InstanceInterfacesField)
	ifsSet := ifsRaw.(*schema.Set)
	ifs := ifsSet.List()
	if len(ifs) > 0 {
		ifaceCreateOptsList := extractInstanceV2InterfaceOptsToListCreate(ifs)
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

	ifsOptsSet := d.Get(InstanceInterfacesField).(*schema.Set)
	ifs := ifsOptsSet.List()
	interfacesMap := extractInstanceV2InterfacesOptsToListRead(ifs)

	var interfacesOptsList []interface{}

	for index, iFace := range interfacesListAPI {
		if len(iFace.IPAssignments) == 0 {
			continue
		}
		for _, assignment := range iFace.IPAssignments {
			subnetID := assignment.SubnetID
			var ok bool
			interfaceOptsMap := make(map[string]interface{})
			var ifsMap map[string]interface{}
			for _, k := range []string{subnetID, iFace.PortID, iFace.NetworkID, string(edgecloudV2.InterfaceTypeExternal)} {
				if k == "external" && !iFace.NetworkDetails.External {
					continue
				}
				ifsMap, ok = interfacesMap[k]
				if ok {
					interfaceOptsMap = ifsMap
					break
				}
			}
			if !ok {
				interfaceOptsMap[NetworkIDField] = iFace.NetworkID
				interfaceOptsMap[SubnetIDField] = assignment.SubnetID
			}
			if index == 0 {
				interfaceOptsMap[IsDefaultField] = true
			}
			interfaceOptsMap[IPAddressField] = assignment.IPAddress.String()
			interfaceOptsMap[OrderField] = index + 1
			interfaceOptsMap[NetworkNameField] = iFace.NetworkDetails.Name
			interfaceOptsMap[PortIDField] = iFace.PortID
			interfacesOptsList = append(interfacesOptsList, interfaceOptsMap)
		}
	}

	if err := d.Set(InstanceInterfacesField, schema.NewSet(ifsOptsSet.F, interfacesOptsList)); err != nil {
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

	if d.HasChange(InstanceInterfacesField) {
		iOldRaw, iNewRaw := d.GetChange(InstanceInterfacesField)
		ifsOldSet, ifsNewSet := iOldRaw.(*schema.Set), iNewRaw.(*schema.Set)
		ifsOldSlice, ifsNewSlice := ifsOldSet.List(), ifsNewSet.List()
		sort.Sort(instanceV2Interfaces(ifsOldSlice))
		sort.Sort(instanceV2Interfaces(ifsNewSlice))
		ifsToDetach := ifsOldSet.Difference(ifsNewSet)
		ifsToAttach := ifsNewSet.Difference(ifsOldSet)

		defaultNewIfsRaw := ifsNewSlice[0]
		defaultNewIfs := defaultNewIfsRaw.(map[string]interface{})
		subnetID := defaultNewIfs[SubnetIDField].(string)
		networkID := defaultNewIfs[NetworkIDField].(string)
		typeIfs := defaultNewIfs[TypeField].(string)
		reservedFixedIPPortID := defaultNewIfs[InstanceReservedFixedIPPortID].(string)

		var indexNewDefaultInOldSlice int
		var maxOrderToDetach int
		var newDefaultExistsInOldSlice bool

		// find max order of iface, that need detach to make iface with field is_default first attached
		oldIfaceMap := extractInstanceV2InterfacesOptsToListRead(ifsOldSlice)
		for _, k := range []string{subnetID, networkID, reservedFixedIPPortID, typeIfs} {
			if iface, ok := oldIfaceMap[k]; ok {
				order := iface[OrderField].(int)
				indexNewDefaultInOldSlice = order - 1
				maxOrderToDetach = order - 1
				newDefaultExistsInOldSlice = true
				break
			}
		}
		if !newDefaultExistsInOldSlice {
			maxOrderToDetach = len(ifsOldSlice)
		}

		// if new is_default iface exists in old state, it is no need detach and attach this iface again
		if newDefaultExistsInOldSlice {
			ifsToDetach.Remove(ifsOldSlice[indexNewDefaultInOldSlice])
			ifsToAttach.Remove(ifsNewSlice[0])
		}

		// choose ifaces that need reattach to make iface with field is_default first attached
		for index, item := range ifsOldSlice {
			if index < maxOrderToDetach {
				ifsToDetach.Add(item)
				if ifsNewSet.Contains(item) {
					ifsToAttach.Add(item)
				}
			} else {
				break
			}
		}

		ifsToDetachList := ifsToDetach.List()
		sort.Sort(instanceV2Interfaces(ifsToDetachList))
		ifsToAttachList := ifsToAttach.List()
		sort.Sort(instanceV2Interfaces(ifsToAttachList))

		for _, item := range ifsToDetachList {
			detachIfs := item.(map[string]interface{})
			if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, detachIfs); err != nil {
				return diag.FromErr(err)
			}
		}
		for _, item := range ifsToAttachList {
			attachIfs := item.(map[string]interface{})
			if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, attachIfs); err != nil {
				return diag.FromErr(err)
			}
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
