package edgecenter

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func resourceLoadBalancerV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLoadBalancerV2Create,
		ReadContext:   resourceLoadBalancerV2Read,
		UpdateContext: resourceLoadBalancerV2Update,
		DeleteContext: resourceLoadBalancerV2Delete,
		Description:   "Represent load balancer without nested listener",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, lbID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(lbID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the load balancer.",
			},
			"flavor": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The flavor or specification of the load balancer to be created.",
			},
			"vip_port_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vip_network_id"},
				Description:   "Attaches the created reserved IP.",
			},
			"vip_network_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vip_port_id"},
				Description:   "Attaches the created network.",
			},
			"vip_subnet_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				RequiredWith: []string{"vip_network_id"},
				Description:  "The ID of the subnet in which to allocate the VIP address for the load balancer.",
			},
			"vip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Load balancer IP address",
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

func resourceLoadBalancerV2Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	opts := &edgecloudV2.LoadbalancerCreateRequest{
		Name:         d.Get("name").(string),
		VipPortID:    d.Get("vip_port_id").(string),
		VipNetworkID: d.Get("vip_network_id").(string),
		VipSubnetID:  d.Get("vip_subnet_id").(string),
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		opts.Metadata = *meta
	}

	lbFlavor := d.Get("flavor").(string)
	if len(lbFlavor) != 0 {
		opts.Flavor = lbFlavor
	}

	createLBTimeout := 5 * time.Minute
	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Loadbalancers.Create, opts, clientV2, createLBTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	lbID := taskResult.Loadbalancers[0]

	d.SetId(lbID)

	resourceLoadBalancerV2Read(ctx, d, m)

	log.Printf("[DEBUG] Finish LoadBalancer creating (%s)", lbID)

	return diags
}

func resourceLoadBalancerV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	lb, _, err := clientV2.Loadbalancers.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("project_id", lb.ProjectID)
	d.Set("region_id", lb.RegionID)
	d.Set("name", lb.Name)
	d.Set("flavor", lb.Flavor.FlavorName)

	if lb.VipAddress != nil {
		d.Set("vip_address", lb.VipAddress.String())
	}

	fields := []string{"vip_network_id", "vip_subnet_id"}
	revertState(d, &fields)

	metadataList, _, err := clientV2.Loadbalancers.MetadataList(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	metadataMap, metadataReadOnly := PrepareMetadata(metadataList)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish LoadBalancer reading")

	return diags
}

func resourceLoadBalancerV2Update(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer updating")
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	if d.HasChange("name") {
		opts := &edgecloudV2.Name{
			Name: d.Get("name").(string),
		}
		if _, _, err = clientV2.Loadbalancers.Rename(ctx, d.Id(), opts); err != nil {
			return diag.FromErr(err)
		}

		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		metadataLB := edgecloudV2.Metadata(*meta)
		_, err = clientV2.Loadbalancers.MetadataUpdate(ctx, d.Id(), &metadataLB)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	log.Println("[DEBUG] Finish LoadBalancer updating")

	return resourceLoadBalancerV2Read(ctx, d, m)
}

func resourceLoadBalancerV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	id := d.Id()
	results, resp, err := clientV2.Loadbalancers.Delete(ctx, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of Load Balancer deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of LoadBalancer deleting")

	return diags
}
