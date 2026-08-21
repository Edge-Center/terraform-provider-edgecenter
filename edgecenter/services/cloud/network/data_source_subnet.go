package network

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const SubnetDataSource = "edgecenter_subnet"

func dataSourceSubnet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubnetRead,
		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.IDField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the subnet. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},
			edgecenter.NameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the subnet.",
				ExactlyOneOf: []string{edgecenter.IDField, edgecenter.NameField},
			},
			edgecenter.MetadataKField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			edgecenter.MetadataKVField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			edgecenter.NetworkIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			edgecenter.EnableDHCPField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			edgecenter.CIDRField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			edgecenter.ConnectToNetworkRouterField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null.",
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
				Elem:        HostRouteSchema(false),
			},
			edgecenter.AllocationPoolsField: {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "A list of allocation pools for DHCP. If omitted but DHCP or gateway settings are changed on update, pools are automatically reassigned.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.StartField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Start IP address.",
						},
						edgecenter.EndField: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "End IP address.",
						},
					},
				},
			},
			edgecenter.GatewayIPField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the gateway for this subnet.",
			},
			edgecenter.MetadataReadOnlyField: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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
		},
	}
}

func dataSourceSubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Subnet reading")

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnet, err := getSubnet(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(subnet.ID)
	_ = d.Set(edgecenter.NameField, subnet.Name)
	_ = d.Set(edgecenter.IDField, subnet.ID)
	_ = d.Set(edgecenter.EnableDHCPField, subnet.EnableDHCP)
	_ = d.Set(edgecenter.CIDRField, subnet.CIDR)
	_ = d.Set(edgecenter.NetworkIDField, subnet.NetworkID)

	metadataReadOnly := edgecenter.PrepareMetadataReadonly(subnet.Metadata)
	if err := d.Set(edgecenter.MetadataReadOnlyField, metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set(edgecenter.DNSNameserversField, dnsNameserversToStringList(subnet.DNSNameservers))
	_ = d.Set(edgecenter.HostRoutesField, hostRoutesToListOfMapsV2(subnet.HostRoutes))
	_ = d.Set(edgecenter.RegionIDField, subnet.RegionID)
	_ = d.Set(edgecenter.ProjectIDField, subnet.ProjectID)
	_ = d.Set(edgecenter.ConnectToNetworkRouterField, true)
	if subnet.GatewayIP != nil {
		_ = d.Set(edgecenter.GatewayIPField, subnet.GatewayIP.String())
	} else {
		_ = d.Set(edgecenter.GatewayIPField, "disable")
		_ = d.Set(edgecenter.ConnectToNetworkRouterField, false)
	}

	allocationPoolsSet := d.Get(edgecenter.AllocationPoolsField).(*schema.Set)

	if err := d.Set(edgecenter.AllocationPoolsField, schema.NewSet(allocationPoolsSet.F, allocationPoolsToListOfMaps(subnet.AllocationPools))); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Subnet reading")

	return nil
}
