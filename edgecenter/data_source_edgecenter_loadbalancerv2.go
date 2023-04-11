package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/loadbalancers"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils/metadata"
)

func dataSourceLoadBalancerV2() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLoadBalancerV2Read,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"vip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vip_port_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_group_id": {
				Type:        schema.TypeString,
				Description: "Load balancer security group ID",
				Computed:    true,
			},
			"metadata_k": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"metadata_kv": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:     schema.TypeList,
				Computed: true,
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

func dataSourceLoadBalancerV2Read(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LoadBalancer reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, LoadBalancersPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)

	metaOpts := &loadbalancers.ListOpts{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		metaOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		meta, err := utils.MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		metaOpts.MetadataKV = meta
	}

	lbs, err := loadbalancers.ListAll(client, *metaOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var lb loadbalancers.LoadBalancer
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

	metadataList, err := metadata.MetadataListAll(client, d.Id())
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

	sgInfo, err := loadbalancers.ListCustomSecurityGroup(client, d.Id()).Extract()
	if err != nil {
		return diag.FromErr(err)
	}
	if len(sgInfo) > 0 {
		d.Set("security_group_id", sgInfo[0].ID)
	}

	log.Println("[DEBUG] Finish LoadBalancer reading")

	return diags
}
