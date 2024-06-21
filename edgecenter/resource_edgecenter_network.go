package edgecenter

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	NetworkDeletingTimeout = 1200 * time.Second
	NetworkCreatingTimeout = 1200 * time.Second
	NetworksPoint          = "networks"
	SharedNetworksPoint    = "availablenetworks"
)

func AllowedNetworkTypes() []string {
	return []string{string(edgecloudV2.VLAN), string(edgecloudV2.VXLAN)}
}

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,
		Description:   "Represent network. A network is a software-defined network in a cloud computing infrastructure",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, NetworkID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(NetworkID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the network.",
			},
			"mtu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum Transmission Unit (MTU) for the network. It determines the maximum packet size that can be transmitted without fragmentation.",
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(AllowedNetworkTypes(), false),
				ForceNew:     true,
				Description:  "'vlan' or 'vxlan' network type is allowed. Default value is 'vxlan'",
			},
			"create_router": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Create external router to the network, default true",
			},
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
			"metadata_map": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Network creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	networkType := d.Get("type").(string)
	createOpts := &edgecloudV2.NetworkCreateRequest{
		Name:         d.Get("name").(string),
		Type:         edgecloudV2.NetworkType(networkType),
		CreateRouter: d.Get("create_router").(bool),
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}

		createOpts.Metadata = *meta
	}

	log.Printf("Create network ops: %+v", createOpts)

	results, _, err := clientV2.Networks.Create(ctx, createOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, &clientV2, taskID, NetworkCreatingTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	createdNetworks, ok := taskInfo.CreatedResources["networks"]
	networkIDs := createdNetworks.([]interface{})

	if !ok || len(networkIDs) == 0 {
		return diag.Errorf("cannot retrieve Network ID from task info: %s", taskID)
	}
	networkID := networkIDs[0].(string)

	log.Printf("[DEBUG] Network id (%s)", networkID)

	d.SetId(networkID)
	resourceNetworkRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Network creating (%s)", networkID)

	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start network reading")
	log.Printf("[DEBUG] Start network reading%s", d.State())
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	networkID := d.Id()
	log.Printf("[DEBUG] Network id = %s", networkID)

	network, _, err := clientV2.Networks.Get(ctx, networkID)
	if err != nil {
		return diag.Errorf("cannot get network with ID: %s. Error: %s", networkID, err)
	}

	d.Set("name", network.Name)
	d.Set("mtu", network.MTU)
	d.Set("type", network.Type)
	d.Set("region_id", network.RegionID)
	d.Set("project_id", network.ProjectID)

	metadataMap, metadataReadOnly := PrepareMetadata(network.Metadata)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	fields := []string{"create_router"}
	revertState(d, &fields)

	log.Println("[DEBUG] Finish network reading")

	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start network updating")
	networkID := d.Id()
	log.Printf("[DEBUG] Volume id = %s", networkID)

	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		newName := &edgecloudV2.Name{
			Name: d.Get("name").(string),
		}
		_, _, err := clientV2.Networks.UpdateName(ctx, networkID, newName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		_, err = clientV2.Networks.MetadataUpdate(ctx, networkID, (*edgecloudV2.Metadata)(meta))
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}
	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish network updating")

	return resourceNetworkRead(ctx, d, m)
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start network deleting")
	networkID := d.Id()
	log.Printf("[DEBUG] Network id = %s", networkID)
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	results, _, err := clientV2.Networks.Delete(ctx, networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)

	task, err := utilV2.WaitAndGetTaskInfo(ctx, &clientV2, taskID, NetworkDeletingTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete network with ID: %s", networkID)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of network deleting")

	return diags
}
