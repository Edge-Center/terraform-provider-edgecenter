package edgecenter

import (
	"context"
	"encoding/json"
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
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the network. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the network. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
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
				Description: "'vlan' or 'vxlan' network type is allowed. Default value is 'vxlan'",
			},
			"external": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"shared": {
				Type:     schema.TypeBool,
				Computed: true,
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

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Network reading")

	var (
		rawNetwork map[string]interface{}
		subs       []edgecloudV2.Subnetwork
		meta       []edgecloudV2.MetadataDetailed
	)

	name := d.Get("name").(string)
	networkID := d.Get("id").(string)
	withDetails := d.Get("shared_with_subnets").(bool)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	fetchNetOpts := fetchNetworksWithSubnetsOptions{
		clientV2: clientV2,
	}

	switch {
	case networkID != "":
		if withDetails {
			fetchNetOpts.fetchOpts = &edgecloudV2.NetworksWithSubnetsOptions{NetworkID: networkID}
			rawNetwork, subs, meta, err = fetchNetworksWithSubnets(ctx, fetchNetOpts)
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			net, _, err := clientV2.Networks.Get(ctx, networkID)
			if err != nil {
				return diag.FromErr(err)
			}

			meta = net.Metadata

			rawNetwork, err = StructToMap(net)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	default:
		if withDetails {
			fetchNetOpts.networkName = name
			rawNetwork, subs, meta, err = fetchNetworksWithSubnets(ctx, fetchNetOpts)
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
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

			nets, _, err := clientV2.Networks.List(ctx, metaOpts)
			if err != nil {
				return diag.FromErr(err)
			}

			network, err := findNetworkByName(name, nets)
			if err != nil {
				return diag.FromErr(err)
			}

			meta = network.Metadata

			rawNetwork, err = StructToMap(network)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId(rawNetwork["id"].(string))
	_ = d.Set("name", rawNetwork["name"])
	_ = d.Set("id", rawNetwork["id"])
	_ = d.Set("mtu", rawNetwork["mtu"])
	_ = d.Set("type", rawNetwork["type"])
	_ = d.Set("region_id", rawNetwork["region_id"])
	_ = d.Set("project_id", rawNetwork["project_id"])
	_ = d.Set("external", rawNetwork["external"])
	_ = d.Set("shared", rawNetwork["shared"])

	if withDetails && len(subs) > 0 {
		if err := d.Set("subnets", prepareSubnets(subs)); err != nil {
			return diag.FromErr(err)
		}
	}

	metadataReadOnly := PrepareMetadataReadonly(meta)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Network reading")

	return nil
}
