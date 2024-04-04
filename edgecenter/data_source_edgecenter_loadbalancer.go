package edgecenter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceLoadBalancer() *schema.Resource {
	return &schema.Resource{
		ReadContext:        dataSourceLoadBalancerRead,
		DeprecationMessage: "!> **WARNING:** This data-source is deprecated and will be removed in the next major version. Use edgecenter_loadbalancerv2 data-source instead",
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
				Description: "The name of the router.",
			},
			"metadata_k": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			"metadata_kv": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}.`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"vip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vip_port_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"listener": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("Available values is '%s' (currently work, other do not work on ed-8), '%s', '%s', '%s'", edgecloudV2.LBPoolProtocolHTTP, edgecloudV2.LBPoolProtocolHTTPS, edgecloudV2.LBPoolProtocolTCP, edgecloudV2.LBPoolProtocolUDP),
						},
						"protocol_port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
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

func dataSourceLoadBalancerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	metaOpts := &edgecloudV2.LoadbalancerListOptions{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		metaOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		typedMetadataKVJson, err := json.Marshal(meta)
		if err != nil {
			return diag.FromErr(err)
		}
		metaOpts.MetadataKV = string(typedMetadataKVJson)
	}

	lbs, _, err := clientV2.Loadbalancers.List(ctx, metaOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var lb edgecloudV2.Loadbalancer
	for _, l := range lbs {
		if l.Name == name {
			lb = l
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("load balancer with name %s not found", name)
	}

	d.SetId(lb.ID)
	d.Set("project_id", lb.ProjectID)
	d.Set("region_id", lb.RegionID)
	d.Set("name", lb.Name)
	d.Set("vip_address", lb.VipAddress.String())
	d.Set("vip_port_id", lb.VipPortID)

	metadataReadOnly := PrepareMetadataReadonly(lb.MetadataDetailed)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	newListeners := make([]map[string]interface{}, len(lb.Listeners))
	for i, l := range lb.Listeners {
		listener, _, err := clientV2.Loadbalancers.ListenerGet(ctx, l.ID)
		if err != nil {
			return diag.FromErr(err)
		}

		newListeners[i] = map[string]interface{}{
			"id":            listener.ID,
			"name":          listener.Name,
			"protocol":      listener.Protocol,
			"protocol_port": listener.ProtocolPort,
		}
	}
	if err := d.Set("listener", newListeners); err != nil {
		diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish LoadBalancer reading")

	return diags
}
