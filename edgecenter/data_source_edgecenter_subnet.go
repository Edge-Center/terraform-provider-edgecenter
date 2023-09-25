package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
)

func dataSourceSubnet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSubnetRead,
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
				Description: "The name of the subnet.",
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
			"network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			"enable_dhcp": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			"connect_to_network_router": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null.",
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

func dataSourceSubnetRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Subnet reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, SubnetPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	networkID := d.Get("network_id").(string)
	subnetsOpts := &subnets.ListOpts{NetworkID: networkID}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		subnetsOpts.MetadataK = metadataK.(string)
	}
	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
		for k, v := range metadataRaw.(map[string]interface{}) {
			typedMetadataKV[k] = v.(string)
		}
		subnetsOpts.MetadataKV = typedMetadataKV
	}

	snets, err := subnets.ListAll(client, *subnetsOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var subnet subnets.Subnet
	for _, sn := range snets {
		if sn.Name == name {
			subnet = sn
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("subnet with name %s not found", name)
	}

	d.SetId(subnet.ID)
	d.Set("name", subnet.Name)
	d.Set("enable_dhcp", subnet.EnableDHCP)
	d.Set("cidr", subnet.CIDR.String())
	d.Set("network_id", subnet.NetworkID)

	metadataReadOnly := PrepareMetadataReadonly(subnet.Metadata)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	dns := make([]string, len(subnet.DNSNameservers))
	for i, ns := range subnet.DNSNameservers {
		dns[i] = ns.String()
	}
	d.Set("dns_nameservers", dns)

	hrs := make([]map[string]string, len(subnet.HostRoutes))
	for i, hr := range subnet.HostRoutes {
		hR := map[string]string{"destination": "", "nexthop": ""}
		hR["destination"] = hr.Destination.String()
		hR["nexthop"] = hr.NextHop.String()
		hrs[i] = hR
	}
	d.Set("host_routes", hrs)
	d.Set("region_id", subnet.RegionID)
	d.Set("project_id", subnet.ProjectID)
	d.Set("gateway_ip", subnet.GatewayIP.String())

	d.Set("connect_to_network_router", true)
	if subnet.GatewayIP == nil {
		d.Set("connect_to_network_router", false)
		d.Set("gateway_ip", "disable")
	}

	log.Println("[DEBUG] Finish Subnet reading")

	return diags
}
