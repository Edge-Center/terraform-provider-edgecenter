package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSubnet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubnetRead,
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			IDField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the subnet. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{IDField, NameField},
			},
			NameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the subnet.",
				ExactlyOneOf: []string{IDField, NameField},
			},
			MetadataKField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			MetadataKVField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			NetworkIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			EnableDHCPField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			CIDRField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			ConnectToNetworkRouterField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null.",
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
			AllocationPoolsField: {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "A list of allocation pools for DHCP. If omitted but DHCP or gateway settings are changed on update, pools are automatically reassigned.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						StartField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Start IP address.",
						},
						EndField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "End IP address.",
						},
					},
				},
			},
			GatewayIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the gateway for this subnet.",
			},
			MetadataReadOnlyField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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
		},
	}
}

func dataSourceSubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Subnet reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnet, err := getSubnet(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subnet.ID)
	_ = d.Set(NameField, subnet.Name)
	_ = d.Set(IDField, subnet.ID)
	_ = d.Set(EnableDHCPField, subnet.EnableDHCP)
	_ = d.Set(CIDRField, subnet.CIDR)
	_ = d.Set(NetworkIDField, subnet.NetworkID)

	metadataReadOnly := PrepareMetadataReadonly(subnet.Metadata)
	if err := d.Set(MetadataReadOnlyField, metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set(DNSNameserversField, dnsNameserversToStringList(subnet.DNSNameservers))
	_ = d.Set(HostRoutesField, hostRoutesToListOfMapsV2(subnet.HostRoutes))
	_ = d.Set(RegionIDField, subnet.RegionID)
	_ = d.Set(ProjectIDField, subnet.ProjectID)
	_ = d.Set(GatewayIPField, subnet.GatewayIP.String())

	allocationPoolsSet := d.Get(AllocationPoolsField).(*schema.Set)

	if err := d.Set(AllocationPoolsField, schema.NewSet(allocationPoolsSet.F, allocationPoolsToListOfMaps(subnet.AllocationPools))); err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set(ConnectToNetworkRouterField, true)
	if subnet.GatewayIP == nil {
		_ = d.Set(ConnectToNetworkRouterField, false)
		_ = d.Set(GatewayIPField, "disable")
	}

	log.Println("[DEBUG] Finish Subnet reading")

	return nil
}
