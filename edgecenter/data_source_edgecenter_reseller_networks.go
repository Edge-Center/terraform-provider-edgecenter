package edgecenter

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	orderByRegexString = `.*\.(asc|desc)`
)

// Maybe move to utils and use for other resources.
var orderByRegex = regexp.MustCompile(orderByRegexString)

func dataSourceResellerNetworksList() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceResellerNetworksRead,
		Description: `
			!!! This data source has been created for resellers and only works with the reseller API key. !!!

	Returns the list of networks with subnet details that are available to the reseller and its clients in all regions.
	If the client_id and project_id parameters are not specified, the network or subnet is not owned by a reseller client or project.`,

		Schema: map[string]*schema.Schema{
			NetworkTypeField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Filter networks by the type of the network (vlan or vxlan).",
				ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VLAN), string(edgecloudV2.VXLAN)}, false),
			},
			OrderByField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Order networks by transmitted fields and directions (name.asc).",
				ValidateFunc: validation.StringMatch(orderByRegex, "must match <any_field_name>.asc|desc"),
			},
			SharedField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Can be used to only show networks with the shared state.",
			},
			MetadataKVField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Filtration query opts, for example, {key = \"value\", key_1 = \"value_1\"}.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			MetadataKField: {
				Type:        schema.TypeSet,
				Description: "Filter by metadata keys. Must be a valid JSON string. \"metadata_k=[\"value\", \"sense\"]\"",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			NetworksField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only reseller networks.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						CreatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The datetime when the network was created.",
						},
						DefaultField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true if the network has is_default attribute.",
						},
						ExternalField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true if the network has router:external attribute.",
						},
						SharedField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true when the network is shared with your project by an external owner.",
						},
						IDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the network.",
						},
						MTUField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The MTU (maximum transmission unit) of the network. Defaults to 1450.",
						},
						NameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network.",
						},
						RegionIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						RegionNameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the region.",
						},
						TypeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the network (vlan, vxlan).",
						},
						SubnetsField: {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: `A list of read-only metadata items, e.g. tags.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									IDField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the subnet.",
									},
									NameField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the subnet.",
									},
									AvailableIPsField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of available IPs in the subnet.",
									},
									TotalIPsField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total number of IPs in the subnet.",
									},
									EnableDHCPField: {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
									},
									HasRouterField: {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether the subnet has a router attached to it.",
									},
									CIDRField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Represents the IP address range of the subnet.",
									},
									DNSNameserversField: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of DNS name servers for the subnet.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									HostRoutesField: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of additional routes to be added to instances that are part of this subnet.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												DestinationField: {
													Type:     schema.TypeString,
													Computed: true,
												},
												NexthopField: {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "IPv4 address to forward traffic to if it's destination IP matches 'destination' CIDR",
												},
											},
										},
									},
									GatewayIPField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The IP address of the gateway for this subnet.",
									},
								},
							},
						},
						CreatorTaskIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The task that created this entity.",
						},
						TaskIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The active task ID this network is locked by.",
						},
						SegmentationIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						UpdatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The datetime when the network was last updated.",
						},

						MetadataField: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: `The metadata of the network.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									KeyField: {
										Type:     schema.TypeString,
										Computed: true,
									},
									ValueField: {
										Type:     schema.TypeString,
										Computed: true,
									},
									ReadOnlyField: {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
						ClientIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the client or null.",
						},
						ProjectIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the project or null.",
						},
					},
				},
			},
		},
	}
}

func dataSourceResellerNetworksRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Start reseller networks reading")

	clientV2, err := InitCloudClient(ctx, d, m, resellerNetworksCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	rnRequest := edgecloudV2.ResellerNetworksListRequest{}

	if v, ok := d.GetOk(NetworkTypeField); ok {
		rnRequest.NetworkType = v.(string)
	}

	if v, ok := d.GetOk(OrderByField); ok {
		rnRequest.OrderBy = v.(string)
	}

	if v, ok := d.GetOk(SharedField); ok {
		rnRequest.Shared = v.(bool)
	}

	if v, ok := d.GetOk(MetadataKVField); ok {
		meta, err := MapInterfaceToMapString(v)
		if err != nil {
			return diag.FromErr(err)
		}

		typedMetadataKVJson, err := json.Marshal(meta)
		if err != nil {
			return diag.FromErr(err)
		}

		rnRequest.MetadataKV = string(typedMetadataKVJson)
	}

	if v, ok := d.GetOk(MetadataKField); ok {
		metaList := v.(*schema.Set).List()

		typedMetadataKJson, err := json.Marshal(metaList)
		if err != nil {
			return diag.FromErr(err)
		}

		rnRequest.MetadataK = string(typedMetadataKJson)
	}

	rnList, _, err := clientV2.ResellerNetworks.List(ctx, &rnRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	// We don't know the ID reseller so we use a simple identifier
	d.SetId("reseller_networks")

	networks := make([]map[string]interface{}, 0, rnList.Count)

	for _, rn := range rnList.Results {
		networks = append(networks, prepareResellerNetwork(rn))
	}

	err = d.Set(NetworksField, networks)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reseller networks reading")

	return nil
}
