package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	BmInstanceDeletingTimeout int = 1200
	BmInstanceCreatingTimeout int = 3600
)

var (
	bmCreateTimeout = time.Second * time.Duration(BmInstanceCreatingTimeout)
	bmDeleteTimeout = time.Second * time.Duration(BmInstanceDeletingTimeout)
)

func resourceBmInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBmInstanceCreate,
		ReadContext:   resourceBmInstanceRead,
		UpdateContext: resourceBmInstanceUpdate,
		DeleteContext: resourceBmInstanceDelete,
		Description:   "Represent baremetal instance",
		Timeouts: &schema.ResourceTimeout{
			Create: &bmCreateTimeout,
		},
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
				Computed:     true,
				Optional:     true,
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
			FlavorIDField: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			InterfaceField: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TypeField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s', '%s', '%s'", edgecloudV2.InterfaceTypeSubnet, edgecloudV2.InterfaceTypeAnySubnet, edgecloudV2.InterfaceTypeExternal, edgecloudV2.InterfaceTypeReservedFixedIP),
						},
						IsParentField: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "If not set will be calculated after creation. Trunk interface always attached first. Can't detach interface if is_parent true. Fields affect only on creation",
						},
						OrderField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Order of attaching interface. Trunk interface always attached first, fields affect only on creation",
						},
						NetworkIDField: {
							Type:        schema.TypeString,
							Description: "required if type is 'subnet' or 'any_subnet'",
							Optional:    true,
						},
						SubnetIDField: {
							Type:        schema.TypeString,
							Description: "required if type is 'subnet'",
							Optional:    true,
						},
						PortIDField: {
							Type:        schema.TypeString,
							Description: "required if type is  'reserved_fixed_ip'",
							Optional:    true,
						},
						ComputedPortIDField: {
							Type:        schema.TypeString,
							Description: "Port ID for all types of network connection",
							Computed:    true,
						},
						// nested map is not supported, in this case, you do not need to use the list for the map
						FipSourceField: {
							Type:        schema.TypeString,
							Description: `Available value is: "new", "existing". Indicates whether the floating IP for this subnet will be new or reused`,
							Optional:    true,
						},
						ExistingFipIDField: {
							Type:        schema.TypeString,
							Description: `If source is existing, the ID of the existing floating IP must be specified`,
							Optional:    true,
						},
						IPAddressField: {
							Type:        schema.TypeString,
							Description: "The address of the network",
							Computed:    true,
						},
					},
				},
			},
			NameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the baremetal instance.",
			},
			NameTemplatesField: {
				Type:          schema.TypeList,
				Optional:      true,
				Deprecated:    "Use name_template instead",
				ConflictsWith: []string{NameTemplateField},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			NameTemplateField: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{NameTemplatesField},
			},
			ImageIDField: {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					ImageIDField,
					ApptemplateIDField,
				},
			},
			ApptemplateIDField: {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					ImageIDField,
					ApptemplateIDField,
				},
			},
			KeypairNameField: {
				Type:     schema.TypeString,
				Optional: true,
			},
			PasswordField: {
				Type:     schema.TypeString,
				Optional: true,
			},
			UsernameField: {
				Type:     schema.TypeString,
				Optional: true,
			},
			MetadataField: {
				Type:          schema.TypeList,
				Optional:      true,
				Deprecated:    "Use metadata_map instead",
				ConflictsWith: []string{MetadataMapField},
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
			MetadataMapField: {
				Type:          schema.TypeMap,
				Optional:      true,
				ConflictsWith: []string{MetadataField},
				Description:   "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			AppConfigField: {
				Type:     schema.TypeMap,
				Optional: true,
			},
			UserDataField: {
				Type:     schema.TypeString,
				Optional: true,
			},
			// computed
			FlavorField: {
				Type:     schema.TypeMap,
				Computed: true,
			},
			StatusField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			VmStateField: {
				Type:     schema.TypeString,
				Computed: true,
			},
			AddressesField: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"net": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"addr": {
										Type:     schema.TypeString,
										Required: true,
									},
									TypeField: {
										Type:     schema.TypeString,
										Required: true,
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

func resourceBmInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start BaremetalInstance creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	diags = validateInterfaceBaremetalOpts(ctx, clientV2, d)
	if diags.HasError() {
		return diags
	}

	ifs := d.Get(InterfaceField).([]interface{})
	// sort interfaces by 'is_parent' at first and by 'order' key to attach it in right order
	sort.Sort(instanceInterfaces(ifs))
	interfaceOptsList := make([]edgecloudV2.BareMetalInterfaceOpts, len(ifs))
	for i, iFace := range ifs {
		raw := iFace.(map[string]interface{})
		interfaceOpts := edgecloudV2.BareMetalInterfaceOpts{
			Type:      edgecloudV2.InterfaceType(raw[TypeField].(string)),
			NetworkID: raw[NetworkIDField].(string),
			SubnetID:  raw[SubnetIDField].(string),
			PortID:    raw[PortIDField].(string),
		}

		if interfaceOpts.NetworkID != "" {
			network, _, err := clientV2.Networks.Get(ctx, interfaceOpts.NetworkID)
			if err != nil {
				return diag.Errorf("Error getting network information: %s", err)
			}
			if network.Type == "vxlan" {
				return diag.Errorf("VxLAN networks are not supported for baremetal instances")
			}
		}

		fipSource := raw[FipSourceField].(string)
		fipID := raw[ExistingFipIDField].(string)
		if fipSource != "" {
			interfaceOpts.FloatingIP = &edgecloudV2.InterfaceFloatingIP{
				Source:             edgecloudV2.FloatingIPSource(fipSource),
				ExistingFloatingID: fipID,
			}
		}
		interfaceOptsList[i] = interfaceOpts
	}

	log.Printf("[DEBUG] Baremetal interfaces: %+v", interfaceOptsList)
	createRequest := edgecloudV2.BareMetalServerCreateRequest{
		Flavor:        d.Get(FlavorIDField).(string),
		ImageID:       d.Get(ImageIDField).(string),
		AppTemplateID: d.Get(ApptemplateIDField).(string),
		KeypairName:   d.Get(KeypairNameField).(string),
		Password:      d.Get(PasswordField).(string),
		Username:      d.Get(UsernameField).(string),
		UserData:      d.Get(UserDataField).(string),
		AppConfig:     d.Get(AppConfigField).(map[string]interface{}),
		Interfaces:    interfaceOptsList,
	}

	name := d.Get(NameField).(string)
	if len(name) > 0 {
		createRequest.Names = []string{name}
	}

	if nameTemplatesRaw, ok := d.GetOk(NameTemplatesField); ok {
		nameTemplates := nameTemplatesRaw.([]interface{})
		if len(nameTemplates) > 0 {
			NameTemp := make([]string, len(nameTemplates))
			for i, nametemp := range nameTemplates {
				NameTemp[i] = nametemp.(string)
			}
			createRequest.NameTemplates = NameTemp
		}
	} else if nameTemplate, ok := d.GetOk(NameTemplateField); ok {
		createRequest.NameTemplates = []string{nameTemplate.(string)}
	}

	if metadata, ok := d.GetOk(MetadataField); ok {
		if len(metadata.([]interface{})) > 0 {
			metadataKV, err := extractKeyValueV2(metadata.([]interface{}))
			if err != nil {
				return diag.FromErr(err)
			}
			metadata, err := MapInterfaceToMapString(metadataKV)
			if err != nil {
				diag.FromErr(err)
			}
			createRequest.Metadata = *metadata
		}
	} else if metadataRaw, ok := d.GetOk(MetadataMapField); ok {
		metadata, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			diag.FromErr(err)
		}
		createRequest.Metadata = *metadata
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Instances.BareMetalCreateInstance, &createRequest, clientV2, bmCreateTimeout)
	if err != nil {
		return diag.Errorf("error creating instance: %s", err)
	}

	instanceID := taskResult.Instances[0]
	log.Printf("[DEBUG] Baremetal Instance id (%s)", instanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(instanceID)
	resourceBmInstanceRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Baremetal Instance creating (%s)", instanceID)

	return diags
}

func resourceBmInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Baremetal Instance reading")
	var diags diag.Diagnostics

	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
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
	d.Set(VmStateField, instance.VMState)

	flavor := make(map[string]interface{}, 4)
	flavor[FlavorIDField] = instance.Flavor.FlavorID
	flavor[FlavorNameField] = instance.Flavor.FlavorName
	flavor[RAMField] = strconv.Itoa(instance.Flavor.RAM)
	flavor[VCPUsField] = strconv.Itoa(instance.Flavor.VCPUS)
	d.Set(FlavorField, flavor)

	interfacesListAPI, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(interfacesListAPI) == 0 {
		return diag.Errorf("interface not found")
	}

	interfacesList, err := convertApiIfaceToTfIface(interfacesListAPI)
	if err != nil {
		return diag.Errorf("convert interface to tf format: %s", err)
	}

	ifs := d.Get(InterfaceField).([]interface{})
	sort.Sort(instanceInterfaces(ifs))

	// If possible, try to get some data from configuration file
	for index, v := range ifs {
		if apiIfaceRaw, ok := SafeGet(interfacesList, index); ok {
			apiIface := apiIfaceRaw.(map[string]interface{})
			tfIface := v.(map[string]interface{})
			tfIfaceType := tfIface[TypeField].(string)

			apiIface[TypeField] = tfIface[TypeField].(string)

			if strings.EqualFold(tfIfaceType, string(edgecloudV2.InterfaceTypeAnySubnet)) {
				apiIface[SubnetIDField] = ""
			}

			if value := tfIface[FipSourceField].(string); value != "" {
				apiIface[FipSourceField] = value
			}
		}
	}

	if err := d.Set(InterfaceField, interfacesList); err != nil {
		return diag.FromErr(err)
	}

	if metadataRaw, ok := d.GetOk(MetadataField); ok {
		metadata := metadataRaw.([]interface{})
		sliced := make([]map[string]string, len(metadata))
		for i, data := range metadata {
			d := data.(map[string]interface{})
			mdata := make(map[string]string, 2)
			md, _, err := clientV2.Instances.MetadataGetItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: d[KeyField].(string)})
			if err != nil {
				return diag.Errorf("cannot get metadata with key: %s. Error: %s", instanceID, err)
			}
			mdata[KeyField] = md.Key
			mdata[ValueField] = md.Value
			sliced[i] = mdata
		}
		d.Set(MetadataField, sliced)
	} else {
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
	}

	addresses := []map[string][]map[string]string{}
	for _, data := range instance.Addresses {
		d := map[string][]map[string]string{}
		netd := make([]map[string]string, len(data))
		for i, iaddr := range data {
			ndata := make(map[string]string, 2)
			ndata[TypeField] = iaddr.Type
			ndata["addr"] = iaddr.Address.String()
			netd[i] = ndata
		}
		d["net"] = netd
		addresses = append(addresses, d)
	}
	if err := d.Set(AddressesField, addresses); err != nil {
		return diag.FromErr(err)
	}

	fields := []string{UserDataField, AppConfigField}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish Instance reading")

	return diags
}

func resourceBmInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Baremetal Instance updating")
	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(NameField) {
		nameTemplates := d.Get(NameTemplatesField).([]interface{})
		nameTemplate := d.Get(NameTemplateField).(string)
		if len(nameTemplate) == 0 && len(nameTemplates) == 0 {
			opts := edgecloudV2.Name{Name: d.Get(NameField).(string)}
			if _, _, err := clientV2.Instances.Rename(ctx, instanceID, &opts); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(MetadataField) {
		omd, nmd := d.GetChange(MetadataField)
		if len(omd.([]interface{})) > 0 {
			for _, data := range omd.([]interface{}) {
				d := data.(map[string]interface{})
				k := d[KeyField].(string)
				_, err = clientV2.Instances.MetadataDeleteItem(ctx, instanceID, &edgecloudV2.MetadataItemOptions{Key: k})
				if err != nil {
					return diag.Errorf("cannot delete metadata key: %s. Error: %s", k, err)
				}
			}
		}
		if len(nmd.([]interface{})) > 0 {
			MetaData := make(edgecloudV2.Metadata)
			for _, data := range nmd.([]interface{}) {
				d := data.(map[string]interface{})
				MetaData[d[KeyField].(string)] = d[ValueField].(string)
			}
			_, err = clientV2.Instances.MetadataCreate(ctx, instanceID, &MetaData)
			if err != nil {
				return diag.Errorf("cannot create metadata. Error: %s", err)
			}
		}
	} else if d.HasChange(MetadataMapField) {
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

	if d.HasChange(InterfaceField) {
		ifsOldRaw, ifsNewRaw := d.GetChange(InterfaceField)

		ifsOld := ifsOldRaw.([]interface{})
		ifsNew := ifsNewRaw.([]interface{})

		diags := validateInterfaceBaremetalOpts(ctx, clientV2, d)
		if diags.HasError() {
			if errSet := d.Set(InterfaceField, ifsOld); errSet != nil {
				return diag.FromErr(errSet)
			}

			return diags
		}

		sort.Sort(instanceInterfaces(ifsNew))

		for index, i := range ifsOld {
			oldIface := i.(map[string]interface{})

			if newIfaceRaw, ok := SafeGet(ifsNew, index); ok {
				newIface := newIfaceRaw.(map[string]interface{})

				if oldIface[IsParentField].(bool) && newIface[IsParentField].(bool) {
					var isTypeEQ = oldIface[IsParentField].(bool) == newIface[IsParentField].(bool)
					var isNetworkIDEQ = oldIface[NetworkIDField].(string) == newIface[NetworkIDField].(string)
					var isSubnetIDEQ = oldIface[SubnetIDField].(string) == newIface[SubnetIDField].(string)
					var isPortIDEQ = oldIface[PortIDField].(string) == newIface[PortIDField].(string)

					if isTypeEQ && isNetworkIDEQ && isSubnetIDEQ && isPortIDEQ {
						continue

					} else {
						if errSet := d.Set(InterfaceField, ifsOld); errSet != nil {
							return diag.FromErr(errSet)
						}

						return diag.Errorf("change first or parent interface is not allowed")
					}
				}
			}

			if isInterfaceContains(oldIface, ifsNew) {
				log.Println("[DEBUG] Skipped, dont need detach")

				continue
			}

			if err = detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, oldIface); err != nil {
				if errSet := d.Set(InterfaceField, ifsOld); errSet != nil {
					return diag.FromErr(errSet)
				}

				return diag.FromErr(err)
			}
		}

		currentIfs, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
		if err != nil {
			if errSet := d.Set(InterfaceField, ifsOld); errSet != nil {
				return diag.FromErr(errSet)
			}
			return diag.FromErr(err)
		}

		for _, i := range ifsNew {
			iface := i.(map[string]interface{})
			if isInterfaceContains(iface, ifsOld) {
				log.Println("[DEBUG] Skipped, dont need attach")
				continue
			}
			if isInterfaceAttachedV2(currentIfs, iface) {
				continue
			}

			iType := edgecloudV2.InterfaceType(iface[TypeField].(string))
			opts := edgecloudV2.InstanceInterface{Type: iType}

			switch iType {
			case edgecloudV2.InterfaceTypeSubnet:
				opts.SubnetID = iface[SubnetIDField].(string)
			case edgecloudV2.InterfaceTypeAnySubnet:
				opts.NetworkID = iface[NetworkIDField].(string)
			case edgecloudV2.InterfaceTypeReservedFixedIP:
				opts.PortID = iface[PortIDField].(string)
			case edgecloudV2.InterfaceTypeExternal:
			}

			log.Printf("[DEBUG] attach interface: %+v", opts)
			if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iface); err != nil {
				if errSet := d.Set(InterfaceField, ifsOld); errSet != nil {
					return diag.FromErr(errSet)
				}
				return diag.FromErr(err)
			}
		}
	}

	d.Set(LastUpdatedField, time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish Instance updating")

	return resourceBmInstanceRead(ctx, d, m)
}

func resourceBmInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Baremetal Instance deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := d.Id()

	var delOpts edgecloudV2.InstanceDeleteOptions
	delOpts.DeleteFloatings = true

	results, _, err := clientV2.Instances.Delete(ctx, instanceID, &delOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, bmDeleteTimeout)
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
