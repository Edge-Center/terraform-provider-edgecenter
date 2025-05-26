package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLoadBalancerV2() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLoadBalancerV2Read,
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
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the load balancer. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the load balancer. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"metadata_k": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			"metadata_kv": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"vip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Load balancer IP address",
			},
			"vip_port_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Attached reserved IP.",
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

func dataSourceLoadBalancerV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	lb, err := getLoadbalancer(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(lb.ID)
	_ = d.Set("project_id", lb.ProjectID)
	_ = d.Set("region_id", lb.RegionID)
	_ = d.Set("name", lb.Name)
	_ = d.Set("id", lb.ID)
	_ = d.Set("vip_address", lb.VipAddress.String())
	_ = d.Set("vip_port_id", lb.VipPortID)

	metadataList, _, err := clientV2.Loadbalancers.MetadataList(ctx, lb.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	metadataReadOnly := make([]map[string]interface{}, 0, len(metadataList))
	if len(metadataList) > 0 {
		for _, metadataItem := range metadataList {
			metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
	}

	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish LoadBalancer reading")

	return nil
}
