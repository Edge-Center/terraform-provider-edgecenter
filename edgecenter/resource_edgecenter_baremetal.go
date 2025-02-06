package edgecenter

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

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
				ForceNew: true,
			},
			"interface": {
				Type:     schema.TypeSet,
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
							Optional:    true,
							Description: "If not set will be calculated after creation. Trunk interface always attached first. Can't detach interface if is_parent true. Fields affect only on creation",
						},
						"is_parent_readonly": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Will be calculated after creation. Can't detach interface if is_parent_readonly true.",
						},
						"order": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Order of attaching interface. Trunk interface always attached first, fields affect only on creation",
						},
						"network_id": {
							Type:         schema.TypeString,
							Description:  "required if type is 'subnet' or 'any_subnet'",
							Optional:     true,
							ValidateFunc: validation.IsUUID,
						},
						"network_name": {
							Type:        schema.TypeString,
							Description: "Name of the network.",
							Computed:    true,
						},
						"subnet_id": {
							Type:         schema.TypeString,
							Description:  "required if type is 'subnet'",
							Optional:     true,
							ValidateFunc: validation.IsUUID,
						},
						"port_id": {
							Type:         schema.TypeString,
							Description:  "required if type is  'reserved_fixed_ip'",
							Optional:     true,
							ValidateFunc: validation.IsUUID,
						},
						"port_id_readonly": {
							Type:        schema.TypeString,
							Description: "Will be calculated after creation",
							Computed:    true,
						},

						// nested map is not supported, in this case, you do not need to use the list for the map
						"fip_source": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  fmt.Sprintf("Indicates whether the floating IP for this subnet will be new or reused. Available value is '%s', '%s'", edgecloudV2.NewFloatingIP, edgecloudV2.ExistingFloatingIP),
							ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.NewFloatingIP), string(edgecloudV2.ExistingFloatingIP)}, true),
						},
						"existing_fip_id": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "If fip_source is \"existing\", the ID of the existing floating IP must be specified",
							ValidateFunc: validation.IsUUID,
						},
						"ip_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Interface IP address",
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
				ForceNew: true,
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

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createRequest := edgecloudV2.BareMetalServerCreateRequest{
		Flavor:        d.Get("flavor_id").(string),
		ImageID:       d.Get("image_id").(string),
		AppTemplateID: d.Get("apptemplate_id").(string),
		KeypairName:   d.Get("keypair_name").(string),
		Password:      d.Get("password").(string),
		Username:      d.Get("username").(string),
		UserData:      d.Get("user_data").(string),
		AppConfig:     d.Get("app_config").(map[string]interface{}),
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

	ifsRaw := d.Get("interface")
	ifsSet := ifsRaw.(*schema.Set)
	ifs := ifsSet.List()
	if len(ifs) > 0 {
		// sort interfaces by 'is_parent' at first and by 'order' key to attach it in right order
		sort.Sort(instanceInterfaces(ifs))
		interfaceOptsList, err := prepareBaremetalInstanceInterfaceCreateOpts(ctx, clientV2, ifs)
		if err != nil {
			return diag.FromErr(err)
		}

		createRequest.Interfaces = interfaceOptsList
	}

	log.Printf("[DEBUG] Baremetal interfaces: %+v", createRequest.Interfaces)

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

	d.Set("region_id", clientV2.Region)
	d.Set("project_id", clientV2.Project)

	instance, resp, err := clientV2.Instances.Get(ctx, instanceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
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

	interfaceAPI := interfacesListAPI[0]

	sourceIfaceMap, err := convertSourceBaremetalInterfaceToMap(interfaceAPI)
	if err != nil {
		return diag.FromErr(err)
	}

	ifsRaw := d.Get("interface")
	ifsSet := ifsRaw.(*schema.Set)
	ifs := ifsSet.List()

	newIfs := make([]interface{}, 0, len(ifs))

	// sort interfaces by 'is_parent' at first and by 'order' key to attach it in right order
	sort.Sort(instanceInterfaces(ifs))

	for _, rawIf := range ifs {
		newIface := maps.Clone(rawIf.(baremetalIfaceMap))
		updateInterfaceState(newIface, sourceIfaceMap)
		newIfs = append(newIfs, newIface)
	}

	if err := d.Set("interface", schema.NewSet(ifsSet.F, newIfs)); err != nil {
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

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
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

		ifsOldScheme := ifsOldRaw.(*schema.Set)
		ifsOld := ifsOldScheme.List()
		ifsNew := ifsNewRaw.(*schema.Set).List()

		for _, iFace := range ifsNew {
			raw := iFace.(baremetalIfaceMap)

			if err := validateBaremetalInterfaceConfig(ctx, raw, clientV2); err != nil {
				if err := d.Set("interface", schema.NewSet(ifsOldScheme.F, ifsOld)); err != nil {
					return diag.FromErr(err)
				}

				return diag.Errorf("validate baremetal interface configuration error: %s", err.Error())
			}
		}

		for _, i := range ifsOld {
			iface := i.(map[string]interface{})
			if isInterfaceContains(iface, ifsNew) {
				tflog.Debug(ctx, "Skipped, dont need detach")
				continue
			}

			if iface["is_parent"].(bool) || iface["is_parent_readonly"].(bool) {
				if err := d.Set("interface", schema.NewSet(ifsOldScheme.F, ifsOld)); err != nil {
					return diag.FromErr(err)
				}

				return diag.Errorf("could not detach trunk interface")
			}

			tflog.Info(ctx, fmt.Sprintf("deattach interface: %+v", iface))
			if err := detachInterfaceFromInstanceV2(ctx, clientV2, instanceID, iface); err != nil {
				if err := d.Set("interface", schema.NewSet(ifsOldScheme.F, ifsOld)); err != nil {
					return diag.FromErr(err)
				}

				return diag.FromErr(err)
			}
		}

		currentIfs, _, err := clientV2.Instances.InterfaceList(ctx, instanceID)
		if err != nil {
			if err := d.Set("interface", schema.NewSet(ifsOldScheme.F, ifsOld)); err != nil {
				return diag.FromErr(err)
			}

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

			tflog.Info(ctx, fmt.Sprintf("attach interface: %+v", opts))
			if err := attachInterfaceToInstanceV2(ctx, clientV2, instanceID, iface); err != nil {
				if err := d.Set("interface", schema.NewSet(ifsOldScheme.F, ifsOld)); err != nil {
					return diag.FromErr(err)
				}

				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("keypair_name") {
		oldKN, _ := d.GetChange("keypair_name")
		d.Set("keypair_name", oldKN.(string))

		return diag.Errorf("changing keypair name for bare-metal instance is prohibited")
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
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
