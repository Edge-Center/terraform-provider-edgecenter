package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	SubnetCreatingTimeout int = 1200
	SubnetPoint               = "subnets"
	disable                   = "disable"
)

func resourceSubnet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubnetCreate,
		ReadContext:   resourceSubnetRead,
		UpdateContext: resourceSubnetUpdate,
		DeleteContext: resourceSubnetDelete,
		Description:   "Represent subnets. Subnetwork is a range of IP addresses in a cloud network. Addresses from this range will be assigned to machines in the cloud",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, subnetID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(subnetID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the subnet.",
			},
			"enable_dhcp": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			"connect_to_network_router": {
				Type:        schema.TypeBool,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null. Default true.",
				Optional:    true,
				Default:     true,
			},
			"dns_nameservers": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "List of DNS name servers for the subnet.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"host_routes": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of additional routes to be added to instances that are part of this subnet.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination": {
							Type:     schema.TypeString,
							Required: true,
						},
						"nexthop": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "IPv4 address to forward traffic to if it's destination IP matches 'destination' CIDR",
						},
					},
				},
			},
			"gateway_ip": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The IP address of the gateway for this subnet.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					IP := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
					if v == disable || IP.MatchString(v) {
						return nil
					}
					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
			},
			"metadata_map": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
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
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceSubnetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Subnet creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	createOpts := &edgecloudV2.SubnetworkCreateRequest{
		Name:                   d.Get("name").(string),
		EnableDHCP:             d.Get("enable_dhcp").(bool),
		NetworkID:              d.Get("network_id").(string),
		ConnectToNetworkRouter: d.Get("connect_to_network_router").(bool),
	}

	cidr := d.Get("cidr").(string)
	if cidr != "" {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.CIDR = cidr
	}

	dnsNameservers := d.Get("dns_nameservers").([]interface{})
	createOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		createOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get("host_routes").([]interface{})
	createOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		createOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gatewayIP := d.Get("gateway_ip").(string)
	gw := net.ParseIP(gatewayIP)
	if gatewayIP == disable {
		createOpts.ConnectToNetworkRouter = false
	} else if gw != nil {
		createOpts.GatewayIP = &gw
	}

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Metadata = *meta
	}

	log.Printf("Create subnet ops: %+v", createOpts)

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Subnetworks.Create, createOpts, clientV2)
	if err != nil {
		return diag.FromErr(err)
	}

	subnetID := taskResult.Subnets[0]

	d.SetId(subnetID)
	resourceSubnetRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Subnet creating (%s)", subnetID)

	return diags
}

func resourceSubnetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet reading")
	log.Printf("[DEBUG] Start subnet reading%s", d.State())
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient
	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	subnet, _, err := clientV2.Subnetworks.Get(ctx, subnetID)
	if err != nil {
		return diag.Errorf("cannot get subnet with ID: %s. Error: %s", subnetID, err)
	}

	d.Set("name", subnet.Name)
	d.Set("enable_dhcp", subnet.EnableDHCP)
	d.Set("cidr", subnet.CIDR)
	d.Set("network_id", subnet.NetworkID)

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

	fields := []string{"connect_to_network_router"}
	revertState(d, &fields)

	if subnet.GatewayIP == nil {
		d.Set("connect_to_network_router", false)
		d.Set("gateway_ip", disable)
	}

	metadataMap, metadataReadOnly := PrepareMetadata(subnet.Metadata)

	if err = d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish subnet reading")

	return diags
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet updating")
	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	updateOpts := &edgecloudV2.SubnetworkUpdateRequest{}

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	updateOpts.EnableDHCP = d.Get("enable_dhcp").(bool)

	// In the structure, the field is mandatory for the ability to transfer the absence of data,
	// if you do not initialize it with a empty list, marshalling will send null and receive a validation error.
	dnsNameservers := d.Get("dns_nameservers").([]interface{})
	updateOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		updateOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get("host_routes").([]interface{})
	updateOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		updateOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("gateway_ip") {
		_, newValue := d.GetChange("gateway_ip")
		if newValue.(string) != disable {
			gatewayIP := net.ParseIP(newValue.(string))
			updateOpts.GatewayIP = &gatewayIP
		}
	}

	_, _, err = clientV2.Subnetworks.Update(ctx, subnetID, updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")
		meta, err := MapInterfaceToMapString(nmd)
		if err != nil {
			return diag.Errorf("metadata wrong fmt. Error: %s", err)
		}

		metaSubnet := edgecloudV2.Metadata(*meta)

		_, err = clientV2.Subnetworks.MetadataUpdate(ctx, subnetID, &metaSubnet)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish subnet updating")

	return resourceSubnetRead(ctx, d, m)
}

func resourceSubnetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	results, resp, err := clientV2.Subnetworks.Delete(ctx, subnetID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of Subnet deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of subnet deleting")

	return diags
}
