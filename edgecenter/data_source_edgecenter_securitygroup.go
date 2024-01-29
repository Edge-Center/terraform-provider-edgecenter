package edgecenter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func dataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecurityGroupRead,
		Description: "Represent SecurityGroups(Firewall)",
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
				Description: "The name of the security group.",
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
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A detailed description of the security group.",
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
			"security_group_rules": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Firewall rules control what inbound(ingress) and outbound(egress) traffic is allowed to enter or leave a Instance. At least one 'egress' rule should be set",
				Set:         secGroupUniqueID,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"direction": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s'", edgecloudV2.SGRuleDirectionIngress, edgecloudV2.SGRuleDirectionEgress),
						},
						"ethertype": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s'", edgecloudV2.EtherTypeIPv4, edgecloudV2.EtherTypeIPv6),
						},
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("Available value is %s", strings.Join(utilV2.SecurityGroupRuleProtocol("").StringList(), ",")),
						},
						"port_range_min": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"port_range_max": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"remote_ip_prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"updated_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start SecurityGroup reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	name := d.Get("name").(string)
	metaOpts := &edgecloudV2.SecurityGroupListOptions{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		metaOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
		for k, v := range metadataRaw.(map[string]interface{}) {
			typedMetadataKV[k] = v.(string)
		}
		typedMetadataKVJson, err := json.Marshal(typedMetadataKV)
		if err != nil {
			return diag.FromErr(err)
		}
		metaOpts.MetadataKV = string(typedMetadataKVJson)
	}

	sgs, _, err := clientV2.SecurityGroups.List(ctx, metaOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var sg edgecloudV2.SecurityGroup
	for _, s := range sgs {
		if s.Name == name {
			sg = s
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("security group with name %s not found", name)
	}

	d.SetId(sg.ID)
	d.Set("project_id", sg.ProjectID)
	d.Set("region_id", sg.RegionID)
	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	metadataReadOnly := make([]map[string]interface{}, 0, len(sg.Metadata))
	if len(sg.Metadata) > 0 {
		for _, metadataItem := range sg.Metadata {
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
	newSgRules := make([]interface{}, len(sg.SecurityGroupRules))
	for i, sgr := range sg.SecurityGroupRules {
		r := make(map[string]interface{})
		r["id"] = sgr.ID
		r["direction"] = string(sgr.Direction)

		r["ethertype"] = ""
		if sgr.EtherType != nil {
			r["ethertype"] = string(*sgr.EtherType)
		}

		r["protocol"] = edgecloudV2.SGRuleProtocolANY
		if sgr.Protocol != nil {
			r["protocol"] = string(*sgr.Protocol)
		}

		r["port_range_max"] = 65535
		if sgr.PortRangeMax != nil {
			r["port_range_max"] = *sgr.PortRangeMax
		}

		r["port_range_min"] = 1
		if sgr.PortRangeMin != nil {
			r["port_range_min"] = *sgr.PortRangeMin
		}

		r["description"] = ""
		if sgr.Description != nil {
			r["description"] = *sgr.Description
		}

		r["remote_ip_prefix"] = ""
		if sgr.RemoteIPPrefix != nil {
			r["remote_ip_prefix"] = *sgr.RemoteIPPrefix
		}

		r["updated_at"] = sgr.UpdatedAt
		r["created_at"] = sgr.CreatedAt

		newSgRules[i] = r
	}

	if err := d.Set("security_group_rules", schema.NewSet(secGroupUniqueID, newSgRules)); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish SecurityGroup reading")

	return diags
}
