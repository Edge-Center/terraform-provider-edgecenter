package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	BmInstanceDeletingTimeout int = 1200
	BmInstanceCreatingTimeout int = 3600
	BmInstancePoint               = "bminstances"
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
				Computed:     true,
				Optional:     true,
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
			"flavor_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"interface": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s', '%s', '%s'", edgecloudV2.InterfaceTypeSubnet, edgecloudV2.InterfaceTypeAnySubnet, edgecloudV2.InterfaceTypeExternal, edgecloudV2.InterfaceTypeReservedFixedIP),
						},
						"is_parent": {
							Type:        schema.TypeBool,
							Computed:    true,
							Optional:    true,
							Description: "If not set will be calculated after creation. Trunk interface always attached first. Can't detach interface if is_parent true. Fields affect only on creation",
						},
						"order": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Order of attaching interface. Trunk interface always attached first, fields affect only on creation",
						},
						"network_id": {
							Type:        schema.TypeString,
							Description: "required if type is 'subnet' or 'any_subnet'",
							Optional:    true,
							Computed:    true,
						},
						"subnet_id": {
							Type:        schema.TypeString,
							Description: "required if type is 'subnet'",
							Optional:    true,
							Computed:    true,
						},
						"port_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "required if type is  'reserved_fixed_ip'",
							Optional:    true,
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
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the baremetal instance.",
			},
			"name_templates": {
				Type:          schema.TypeList,
				Optional:      true,
				Deprecated:    "Use name_template instead",
				ConflictsWith: []string{"name_template"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"name_template": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_templates"},
			},
			"image_id": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"image_id",
					"apptemplate_id",
				},
			},
			"apptemplate_id": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"image_id",
					"apptemplate_id",
				},
			},
			"keypair_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
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
			"app_config": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// computed
			"flavor": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vm_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"addresses": {
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
									"type": {
										Type:     schema.TypeString,
										Required: true,
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

func resourceBmInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start BaremetalInstance creating")
	var diags diag.Diagnostics
	config := m.(*Config)

	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	ifs := d.Get("interface").([]interface{})
	// sort interfaces by 'is_parent' at first and by 'order' key to attach it in right order
	sort.Sort(instanceInterfaces(ifs))
	interfaceOptsList := make([]edgecloudV2.BareMetalInterfaceOpts, len(ifs))
	for i, iFace := range ifs {
		raw := iFace.(map[string]interface{})
		interfaceOpts := edgecloudV2.BareMetalInterfaceOpts{
			Type:      edgecloudV2.InterfaceType(raw["type"].(string)),
			NetworkID: raw["network_id"].(string),
			SubnetID:  raw["subnet_id"].(string),
			PortID:    raw["port_id"].(string),
		}

		fipSource := raw["fip_source"].(string)
		fipID := raw["existing_fip_id"].(string)
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
		Flavor:        d.Get("flavor_id").(string),
		ImageID:       d.Get("image_id").(string),
		AppTemplateID: d.Get("apptemplate_id").(string),
		KeypairName:   d.Get("keypair_name").(string),
		Password:      d.Get("password").(string),
		Username:      d.Get("username").(string),
		UserData:      d.Get("user_data").(string),
		AppConfig:     d.Get("app_config").(map[string]interface{}),
		Interfaces:    interfaceOptsList,
	}

	name := d.Get("name").(string)
	if len(name) > 0 {
		createRequest.Names = []string{name}
	}

	if nameTemplatesRaw, ok := d.GetOk("name_templates"); ok {
		nameTemplates := nameTemplatesRaw.([]interface{})
		if len(nameTemplates) > 0 {
			NameTemp := make([]string, len(nameTemplates))
			for i, nametemp := range nameTemplates {
				NameTemp[i] = nametemp.(string)
			}
			createRequest.NameTemplates = NameTemp
		}
	} else if nameTemplate, ok := d.GetOk("name_template"); ok {
		createRequest.NameTemplates = []string{nameTemplate.(string)}
	}

	if metadata, ok := d.GetOk("metadata"); ok {
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
	} else if metadataRaw, ok := d.GetOk("metadata_map"); ok {
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
	config := m.(*Config)
	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)

	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID
	d.Set("region_id", regionID)
	d.Set("project_id", projectID)

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

	interfacesListAPI, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(interfacesListAPI) == 0 {
		return diag.Errorf("interface not found")
	}

	ifs := d.Get("interface").([]interface{})
	orderedInterfacesMap := extractInstanceInterfaceToListReadV2(ifs)
	if err != nil {
		return diag.FromErr(err)
	}

	var interfacesList []interface{}
	for _, iFace := range interfacesListAPI {
		if len(iFace.IPAssignments) == 0 {
			continue
		}
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
			if iFace.SubPorts != nil {
				i["is_parent"] = true
			}
			if interfaceOpts.InstanceInterface.FloatingIP != nil {
				i["fip_source"] = interfaceOpts.InstanceInterface.FloatingIP.Source
				i["existing_fip_id"] = interfaceOpts.InstanceInterface.FloatingIP.ExistingFloatingID
			}
			i["ip_address"] = assignment.IPAddress.String()

			interfacesList = append(interfacesList, i)
		}
		for _, iFaceSubPort := range iFace.SubPorts {
			for _, assignmentSubPort := range iFaceSubPort.IPAssignments {
				assignmentSubnetID := assignmentSubPort.SubnetID

				var interfaceOpts OrderedInterfaceOpts
				var orderedInterfaceOpts OrderedInterfaceOpts
				var ok bool

				for _, k := range []string{assignmentSubnetID, iFaceSubPort.PortID, iFaceSubPort.NetworkID, string(edgecloudV2.InterfaceTypeExternal)} {
					if orderedInterfaceOpts, ok = orderedInterfacesMap[k]; ok {
						interfaceOpts = orderedInterfaceOpts
						break
					}
				}

				i := make(map[string]interface{})

				i["type"] = interfaceOpts.InstanceInterface.Type
				i["order"] = interfaceOpts.Order
				i["network_id"] = iFaceSubPort.NetworkID
				i["subnet_id"] = assignmentSubnetID
				i["port_id"] = iFaceSubPort.PortID
				i["is_parent"] = false
				if interfaceOpts.InstanceInterface.FloatingIP != nil {
					i["fip_source"] = interfaceOpts.InstanceInterface.FloatingIP.Source
					i["existing_fip_id"] = interfaceOpts.InstanceInterface.FloatingIP.ExistingFloatingID
				}
				i["ip_address"] = assignmentSubPort.IPAddress.String()

				interfacesList = append(interfacesList, i)
			}
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
				return diag.Errorf("cannot get metadata with key: %s. Error: %s", instanceID, err)
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

	fields := []string{"user_data", "app_config"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish Instance reading")

	return diags
}

func resourceBmInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Baremetal Instance updating")
	instanceID := d.Id()
	log.Printf("[DEBUG] Instance id = %s", instanceID)
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

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
		if len(nmd.([]interface{})) > 0 {
			MetaData := make(edgecloudV2.Metadata)
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
		ifsOldRaw, ifsNewRaw := d.GetChange("interface")

		ifsOld := ifsOldRaw.([]interface{})
		ifsNew := ifsNewRaw.([]interface{})

		for _, i := range ifsOld {
			iface := i.(map[string]interface{})
			if isInterfaceContains(iface, ifsNew) {
				log.Println("[DEBUG] Skipped, dont need detach")
				continue
			}

			if iface["is_parent"].(bool) {
				return diag.Errorf("could not detach trunk interface")
			}
			if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iface); err != nil {
				return diag.FromErr(err)
			}
		}

		currentIfs, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
		if err != nil {
			return diag.FromErr(err)
		}

		sort.Sort(instanceInterfaces(ifsNew))
		for _, i := range ifsNew {
			iface := i.(map[string]interface{})
			if isInterfaceContains(iface, ifsOld) {
				log.Println("[DEBUG] Skipped, dont need attach")
				continue
			}
			if isInterfaceAttachedV2(currentIfs, iface) {
				continue
			}

			iType := edgecloudV2.InterfaceType(iface["type"].(string))
			opts := edgecloudV2.InstanceInterface{Type: iType}
			switch iType {
			case edgecloudV2.InterfaceTypeSubnet:
				opts.SubnetID = iface["subnet_id"].(string)
			case edgecloudV2.InterfaceTypeAnySubnet:
				opts.NetworkID = iface["network_id"].(string)
			case edgecloudV2.InterfaceTypeReservedFixedIP:
				opts.PortID = iface["port_id"].(string)
			case edgecloudV2.InterfaceTypeExternal:
			}

			log.Printf("[DEBUG] attach interface: %+v", opts)
			if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iface); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish Instance updating")

	return resourceBmInstanceRead(ctx, d, m)
}

func resourceBmInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Baremetal Instance deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID
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
