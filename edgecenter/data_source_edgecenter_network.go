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

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNetworkRead,
		Description: "Represent network. A network is a software-defined network in a cloud computing infrastructure",
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
			"shared_with_subnets": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Get shared networks with details of subnets.",
			},
			"mtu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum Transmission Unit (MTU) for the network. It determines the maximum packet size that can be transmitted without fragmentation.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("The type of the network. Available values are `%s` or `%s`. Default value is `vxlan`.", edgecloudV2.VLAN, edgecloudV2.VXLAN),
			},
			"external": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "`true` if the network has router:external attribute.",
			},
			"shared": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "`true` if the network has router:external attribute.",
			},
			"subnets": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the subnet.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the subnet.",
						},
						"available_ips": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of available IPs in the subnet.",
						},
						"total_ips": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The total number of IPs in the subnet.",
						},
						"enable_dhcp": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
						},
						"has_router": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether the subnet has a router attached to it.",
						},
						"cidr": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Represents the IP address range of the subnet.",
						},
						"dns_nameservers": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of DNS name servers for the subnet.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"host_routes": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of additional routes to be added to instances that are part of this subnet.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"destination": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"nexthop": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "IPv4 address to forward traffic to if it's destination IP matches 'destination' CIDR",
									},
								},
							},
						},
						"gateway_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IP address of the gateway for this subnet.",
						},
					},
				},
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
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "This parameter represents a key in the metadata.",
						},
						"value": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "This parameter represents the value associated with the key in the metadata.",
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

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Network reading")
	var diags diag.Diagnostics
	config := m.(*Config)

	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	metaOpts := &edgecloudV2.NetworkListOptions{}

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

	var (
		withDetails = d.Get("shared_with_subnets").(bool)
		rawNetwork  map[string]interface{}
		subs        []edgecloudV2.Subnetwork
		meta        []edgecloudV2.MetadataDetailed
	)

	if !withDetails {
		nets, _, err := clientV2.Networks.List(ctx, metaOpts)
		if err != nil {
			return diag.FromErr(err)
		}
		network, found := findNetworkByName(name, nets)
		if !found {
			return diag.Errorf("network with name %s not found. you can try to set 'shared_with_subnets' parameter", name)
		}
		meta = network.Metadata
		rawNetwork, err = StructToMap(network)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		nets, _, err := clientV2.Networks.ListNetworksWithSubnets(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
		sharedNetwork, found := findSharedNetworkByName(name, nets)
		if !found {
			return diag.Errorf("shared network with name %s not found", name)
		}
		subs = sharedNetwork.Subnets
		rawNetwork, err = StructToMap(sharedNetwork)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(rawNetwork["id"].(string))
	d.Set("name", rawNetwork["name"])
	d.Set("mtu", rawNetwork["mtu"])
	d.Set("type", rawNetwork["type"])
	d.Set("region_id", rawNetwork["region_id"])
	d.Set("project_id", rawNetwork["project_id"])
	d.Set("external", rawNetwork["external"])
	d.Set("shared", rawNetwork["shared"])
	if withDetails {
		if len(subs) > 0 {
			if err := d.Set("subnets", prepareSubnets(subs)); err != nil {
				return diag.FromErr(err)
			}
		}
	} else {
		metadataReadOnly := PrepareMetadataReadonly(meta)
		if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Println("[DEBUG] Finish Network reading")

	return diags
}
