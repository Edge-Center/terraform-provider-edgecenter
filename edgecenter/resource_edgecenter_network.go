package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"
)

const (
	NetworkDeleting        int = 1200
	NetworkCreatingTimeout int = 1200
	NetworksPoint              = "networks"
	SharedNetworksPoint        = "availablenetworks"
)

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
				Optional:    true,
				Computed:    true,
				Description: "Maximum Transmission Unit (MTU) for the network. It determines the maximum packet size that can be transmitted without fragmentation.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "'vlan' or 'vxlan' network type is allowed. Default value is 'vxlan'",
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
	provider := config.Provider

	client, err := CreateClient(provider, d, NetworksPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := networks.CreateOpts{
		Name:         d.Get("name").(string),
		Type:         d.Get("type").(string),
		CreateRouter: d.Get("create_router").(bool),
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := utils.MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}

		createOpts.Metadata = meta
	}

	log.Printf("Create network ops: %+v", createOpts)
	results, err := networks.Create(client, createOpts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	networkID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, NetworkCreatingTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		NetworkID, err := networks.ExtractNetworkIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve Network ID from task info: %w", err)
		}
		return NetworkID, nil
	},
	)
	log.Printf("[DEBUG] Network id (%s)", networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(networkID.(string))
	resourceNetworkRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Network creating (%s)", networkID)

	return diags
}

func resourceNetworkRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start network reading")
	log.Printf("[DEBUG] Start network reading%s", d.State())
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider
	networkID := d.Id()
	log.Printf("[DEBUG] Network id = %s", networkID)

	client, err := CreateClient(provider, d, NetworksPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	network, err := networks.Get(client, networkID).Extract()
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
	provider := config.Provider
	client, err := CreateClient(provider, d, NetworksPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		_, err := networks.Update(client, networkID, networks.UpdateOpts{Name: newName}).Extract()
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := utils.MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		err = metadata.ResourceMetadataReplace(client, networkID, meta).Err
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}
	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish network updating")

	return resourceNetworkRead(ctx, d, m)
}

func resourceNetworkDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start network deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider
	networkID := d.Id()
	log.Printf("[DEBUG] Network id = %s", networkID)

	client, err := CreateClient(provider, d, NetworksPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	results, err := networks.Delete(client, networkID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, NetworkDeleting, func(task tasks.TaskID) (interface{}, error) {
		_, err := networks.Get(client, networkID).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete network with ID: %s", networkID)
		}
		var errDefault404 edgecloud.Default404Error
		if errors.As(err, &errDefault404) {
			return nil, nil
		}
		return nil, fmt.Errorf("extracting Network resource error: %w", err)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of network deleting")

	return diags
}
