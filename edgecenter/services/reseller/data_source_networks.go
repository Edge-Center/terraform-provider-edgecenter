package reseller

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	cloudnetwork "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/cloud/network"
)

const (
	ResellerNetworksDataSource = "edgecenter_reseller_networks"

	orderByRegexString = `.*\.(asc|desc)`
)

var orderByRegex = regexp.MustCompile(orderByRegexString)

func dataSourceResellerNetworksList() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceResellerNetworksRead,
		Description: `
			!!! This data source has been created for resellers and only works with the reseller API key. !!!

	Returns the list of networks with subnet details that are available to the reseller and its clients in all regions.
	If the client_id and project_id parameters are not specified, the network or subnet is not owned by a reseller client or project.`,

		Schema: map[string]*schema.Schema{
			edgecenter.NetworkTypeField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Filter networks by the type of the network (vlan or vxlan).",
				ValidateFunc: validation.StringInSlice([]string{string(edgecloudV2.VLAN), string(edgecloudV2.VXLAN)}, false),
			},
			edgecenter.OrderByField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Order networks by transmitted fields and directions (name.asc).",
				ValidateFunc: validation.StringMatch(orderByRegex, "must match <any_field_name>.asc|desc"),
			},
			edgecenter.SharedField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Can be used to only show networks with the shared state.",
			},
			edgecenter.MetadataKVField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Filtration query opts, for example, {key = \"value\", key_1 = \"value_1\"}.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			edgecenter.MetadataKField: {
				Type:        schema.TypeSet,
				Description: "Filter by metadata keys. Must be a valid JSON string. \"metadata_k=[\"value\", \"sense\"]\"",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			edgecenter.NetworksField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only reseller networks.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.CreatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The datetime when the network was created.",
						},
						edgecenter.DefaultField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true if the network has is_default attribute.",
						},
						edgecenter.ExternalField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true if the network has router:external attribute.",
						},
						edgecenter.SharedField: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "true when the network is shared with your project by an external owner.",
						},
						edgecenter.IDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the network.",
						},
						edgecenter.MTUField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The MTU (maximum transmission unit) of the network. Defaults to 1450.",
						},
						edgecenter.NameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network.",
						},
						edgecenter.RegionIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						edgecenter.RegionNameField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the region.",
						},
						edgecenter.TypeField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the network (vlan, vxlan).",
						},
						edgecenter.SubnetsField: {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: `A list of read-only metadata items, e.g. tags.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									edgecenter.IDField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the subnet.",
									},
									edgecenter.NameField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the subnet.",
									},
									edgecenter.AvailableIPsField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of available IPs in the subnet.",
									},
									edgecenter.TotalIPsField: {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total number of IPs in the subnet.",
									},
									edgecenter.EnableDHCPField: {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
									},
									edgecenter.HasRouterField: {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether the subnet has a router attached to it.",
									},
									edgecenter.CIDRField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Represents the IP address range of the subnet.",
									},
									edgecenter.DNSNameserversField: {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of DNS name servers for the subnet.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									edgecenter.HostRoutesField: {
										Type:        schema.TypeSet,
										Computed:    true,
										Description: "Set of additional routes to be added to instances that are part of this subnet.",
										Elem:        cloudnetwork.HostRouteSchema(false),
									},
									edgecenter.GatewayIPField: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The IP address of the gateway for this subnet.",
									},
								},
							},
						},
						edgecenter.CreatorTaskIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The task that created this entity.",
						},
						edgecenter.TaskIDField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The active task ID this network is locked by.",
						},
						edgecenter.SegmentationIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the region.",
						},
						edgecenter.UpdatedAtField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The datetime when the network was last updated.",
						},

						edgecenter.MetadataField: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: `The metadata of the network.`,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									edgecenter.KeyField: {
										Type:     schema.TypeString,
										Computed: true,
									},
									edgecenter.ValueField: {
										Type:     schema.TypeString,
										Computed: true,
									},
									edgecenter.ReadOnlyField: {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
						edgecenter.ClientIDField: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The ID of the client or null.",
						},
						edgecenter.ProjectIDField: {
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, resellerNetworksCloudClientConf())
	if err != nil {
		return diag.FromErr(err)
	}

	rnRequest := edgecloudV2.ResellerNetworksListRequest{}

	if v, ok := d.GetOk(edgecenter.NetworkTypeField); ok {
		rnRequest.NetworkType = v.(string)
	}

	if v, ok := d.GetOk(edgecenter.OrderByField); ok {
		rnRequest.OrderBy = v.(string)
	}

	if v, ok := d.GetOk(edgecenter.SharedField); ok {
		rnRequest.Shared = v.(bool)
	}

	if v, ok := d.GetOk(edgecenter.MetadataKVField); ok {
		meta, err := edgecenter.MapInterfaceToMapString(v)
		if err != nil {
			return diag.FromErr(err)
		}

		typedMetadataKVJson, err := json.Marshal(meta)
		if err != nil {
			return diag.FromErr(err)
		}

		rnRequest.MetadataKV = string(typedMetadataKVJson)
	}

	if v, ok := d.GetOk(edgecenter.MetadataKField); ok {
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

	d.SetId("reseller_networks")

	networks := make([]map[string]interface{}, 0, rnList.Count)

	for _, rn := range rnList.Results {
		networks = append(networks, prepareResellerNetwork(rn))
	}

	err = d.Set(edgecenter.NetworksField, networks)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Finish reseller networks reading")

	return nil
}

func prepareResellerNetwork(rn edgecloudV2.ResellerNetwork) map[string]interface{} {
	network := make(map[string]interface{})

	network[edgecenter.CreatedAtField] = rn.CreatedAt
	network[edgecenter.DefaultField] = rn.Default
	network[edgecenter.ExternalField] = rn.External
	network[edgecenter.SharedField] = rn.Shared
	network[edgecenter.IDField] = rn.ID
	network[edgecenter.MTUField] = rn.MTU
	network[edgecenter.NameField] = rn.Name
	network[edgecenter.RegionIDField] = rn.RegionID
	network[edgecenter.RegionNameField] = rn.Region
	network[edgecenter.TypeField] = rn.Type
	network[edgecenter.SubnetsField] = cloudnetwork.PrepareSubnets(rn.Subnets)
	network[edgecenter.CreatorTaskIDField] = rn.CreatorTaskID
	network[edgecenter.TaskIDField] = rn.TaskID
	network[edgecenter.SegmentationIDField] = rn.SegmentationID
	network[edgecenter.UpdatedAtField] = rn.UpdatedAt
	network[edgecenter.ClientIDField] = rn.ClientID
	network[edgecenter.ProjectIDField] = rn.RegionID
	network[edgecenter.MetadataField] = edgecenter.PrepareMetadataReadonly(rn.Metadata)

	return network
}

func resellerNetworksCloudClientConf() *edgecenter.CloudClientConf {
	return &edgecenter.CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
}
