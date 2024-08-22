package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"
	"slices"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	FloatingIPsPoint        = "floatingips"
	FloatingIPCreateTimeout = 1200 * time.Second
	FloatingIPDeleteTimeout = 1200 * time.Second
)

func resourceFloatingIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFloatingIPCreate,
		ReadContext:   resourceFloatingIPRead,
		UpdateContext: resourceFloatingIPUpdate,
		DeleteContext: resourceFloatingIPDelete,
		Description: `A floating IP is a static IP address that can be associated with one of your instances or loadbalancers, 
allowing it to have a static public IP address. The floating IP can be re-associated to any other instance in the same datacenter.`,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, fipID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(fipID)

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
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The floating IP address assigned to the resource.",
			},
			"port_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID (uuid) of the network port that the floating IP is associated with.",
			},
			"instance_port_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the network port of the instance that the floating IP is associated with.",
			},
			"load_balancers_port_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the network port of the load balancer that the floating IP is associated with.",
			},
			"instance_id_attached_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the instance, that the floating IP is associated with.",
			},
			"load_balancers_id_attached_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the loadbalancer, that the floating IP associated with",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the floating IP. Can be 'DOWN' or 'ACTIVE'.",
			},
			"fixed_ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The fixed (reserved) IP address that is associated with the floating IP.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
			},
			"router_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID (uuid) of the router that the floating IP is associated with.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the floating IP was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp when the floating IP was updated.",
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

func resourceFloatingIPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := &edgecloudV2.FloatingIPCreateRequest{
		PortID:         d.Get("port_id").(string),
		FixedIPAddress: net.ParseIP(d.Get("fixed_ip_address").(string)),
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		opts.Metadata = *meta
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Floatingips.Create, opts, clientV2, FloatingIPCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	floatingIPID := taskResult.FloatingIPs[0]

	log.Printf("[DEBUG] FloatingIP id (%s)", floatingIPID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(floatingIPID)
	resourceFloatingIPRead(ctx, d, m)

	log.Printf("[DEBUG] Finish FloatingIP creating (%s)", floatingIPID)

	return diags
}

func resourceFloatingIPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	floatingIPs, response, err := clientV2.Floatingips.List(ctx)
	if err != nil {
		log.Printf("[WARN] Error while GET list floatingIPs. StstusCode: %d.", response.StatusCode)
		return diag.FromErr(err)
	}
	index := slices.IndexFunc(floatingIPs, func(f edgecloudV2.FloatingIP) bool {
		return f.ID == d.Id()
	})
	if index == -1 {
		log.Printf("[WARN] Removing floating ip %s because resource doesn't exist anymore", d.Id())
		d.SetId("")
		return diag.FromErr(fmt.Errorf("could not find a floatingIP with id: %s", d.Id()))
	}
	floatingIP := floatingIPs[index]
	if floatingIP.FixedIPAddress != nil {
		d.Set("fixed_ip_address", floatingIP.FixedIPAddress.String())
	} else {
		d.Set("fixed_ip_address", "")
	}
	d.Set("project_id", floatingIP.ProjectID)
	d.Set("region_id", floatingIP.RegionID)
	d.Set("status", floatingIP.Status)
	if floatingIP.Instance.ID != "" {
		d.Set("instance_id_attached_to", floatingIP.Instance.ID)
		d.Set("instance_port_id", floatingIP.PortID)
	}
	if floatingIP.Loadbalancer.ID != "" {
		d.Set("load_balancer_id_attached_to", floatingIP.Loadbalancer.ID)
		d.Set("load_balancer_port_id", floatingIP.PortID)
	}
	d.Set("port_id", floatingIP.PortID)
	d.Set("router_id", floatingIP.RouterID)
	d.Set("floating_ip_address", floatingIP.FloatingIPAddress)

	metadataMap, metadataReadOnly := PrepareMetadata(floatingIP.Metadata)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish FloatingIP reading")

	return diags
}

func resourceFloatingIPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP updating")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("fixed_ip_address", "port_id") {
		oldFixedIP, newFixedIP := d.GetChange("fixed_ip_address")
		oldPortID, newPortID := d.GetChange("port_id")
		if oldPortID.(string) != "" || oldFixedIP.(string) != "" {
			_, _, err := clientV2.Floatingips.UnAssign(ctx, d.Id())
			if err != nil {
				return diag.FromErr(err)
			}
		}
		opts := &edgecloudV2.AssignFloatingIPRequest{}
		if portID := newPortID.(string); portID != "" {
			opts.PortID = portID
			if fixedIP := newFixedIP.(string); fixedIP != "" {
				opts.FixedIPAddress = net.ParseIP(fixedIP)
			}
			_, _, err = clientV2.Floatingips.Assign(ctx, d.Id(), opts)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		meta, err := MapInterfaceToMapString(nmd.(map[string]interface{}))
		if err != nil {
			return diag.Errorf("cannot get metadata. Error: %s", err)
		}

		metaChanged := edgecloudV2.Metadata(*meta)
		_, err = clientV2.Floatingips.MetadataUpdate(ctx, d.Id(), &metaChanged)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	return resourceFloatingIPRead(ctx, d, m)
}

func resourceFloatingIPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()

	results, _, err := clientV2.Floatingips.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, FloatingIPDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete floating ip with ID: %s", id)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of FloatingIP deleting")

	return diags
}
