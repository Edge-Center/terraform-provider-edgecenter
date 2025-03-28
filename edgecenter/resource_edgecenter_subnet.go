package edgecenter

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	SubnetCreatingTimeout = 1200 * time.Second
	SubnetPoint           = "subnets"
	disable               = "disable"
)

func resourceSubnet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSubnetCreate,
		ReadContext:   resourceSubnetRead,
		UpdateContext: resourceSubnetUpdate,
		DeleteContext: resourceSubnetDelete,
		Description:   "Represent subnets. Subnetwork is a range of IP addresses in a cloud network. Addresses from this range will be assigned to machines in the cloud.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, subnetID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set(ProjectIDField, projectID)
				d.Set(RegionIDField, regionID)
				d.SetId(subnetID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the subnet.",
			},
			EnableDHCPField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			CIDRField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			NetworkIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			ConnectToNetworkRouterField: {
				Type:        schema.TypeBool,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null. Default true.",
				Optional:    true,
				Default:     true,
			},
			DNSNameserversField: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of DNS name servers for the subnet.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			HostRoutesField: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of additional routes to be added to instances that are part of this subnet.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						DestinationField: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The CIDR of the destination IPv4 subnet",
						},
						NexthopField: {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsIPAddress,
							Description:  "IPv4 address to forward traffic to if it's destination IP matches 'destination' CIDR",
						},
					},
				},
			},
			GatewayIPField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The IP address of the gateway for this subnet. The subnet will be recreated if the gateway IP is changed.",
				ValidateFunc: validateSubnetGatewayIP,
			},
			AllocationPoolsField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A list of allocation pools for DHCP. If omitted but DHCP or gateway settings are changed on update, pools are automatically reassigned.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						StartField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Start IP address.",
							ValidateFunc: validation.IsIPAddress,
						},
						EndField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "End IP address.",
							ValidateFunc: validation.IsIPAddress,
						},
					},
				},
			},
			MetadataMapField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
			LastUpdatedField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceSubnetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Subnet creating")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := &edgecloudV2.SubnetworkCreateRequest{
		Name:                   d.Get(NameField).(string),
		EnableDHCP:             d.Get(EnableDHCPField).(bool),
		NetworkID:              d.Get(NetworkIDField).(string),
		ConnectToNetworkRouter: d.Get(ConnectToNetworkRouterField).(bool),
	}

	rawAPs, ok := d.GetOk(AllocationPoolsField)
	if ok {
		createOpts.AllocationPools = prepareSubnetAllocationPools(rawAPs.(*schema.Set).List())
	}

	cidr := d.Get(CIDRField).(string)
	if cidr != "" {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.CIDR = cidr
	}

	dnsNameservers := d.Get(DNSNameserversField).([]interface{})
	createOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		createOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get(HostRoutesField).([]interface{})
	createOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		createOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gatewayIP := d.Get(GatewayIPField).(string)
	gw := net.ParseIP(gatewayIP)
	if gatewayIP == disable {
		createOpts.ConnectToNetworkRouter = false
	} else if gw != nil {
		createOpts.GatewayIP = &gw
	}

	if metadataRaw, ok := d.GetOk(MetadataMapField); ok {
		meta, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.Metadata = *meta
	}

	log.Printf("Create subnet ops: %+v", createOpts)

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Subnetworks.Create, createOpts, clientV2, SubnetCreatingTimeout)
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

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)
	subnet, _, err := clientV2.Subnetworks.Get(ctx, subnetID)
	if err != nil {
		return diag.Errorf("cannot get subnet with ID: %s. Error: %s", subnetID, err)
	}

	d.Set(NameField, subnet.Name)
	d.Set(EnableDHCPField, subnet.EnableDHCP)
	d.Set(CIDRField, subnet.CIDR)
	d.Set(NetworkIDField, subnet.NetworkID)

	dns := make([]string, len(subnet.DNSNameservers))
	for i, ns := range subnet.DNSNameservers {
		dns[i] = ns.String()
	}
	d.Set(DNSNameserversField, dns)

	hrs := make([]map[string]string, len(subnet.HostRoutes))
	for i, hr := range subnet.HostRoutes {
		hR := map[string]string{DestinationField: "", NexthopField: ""}
		hR[DestinationField] = hr.Destination.String()
		hR[NexthopField] = hr.NextHop.String()
		hrs[i] = hR
	}

	d.Set(HostRoutesField, hrs)

	allocationPoolsSet := d.Get(AllocationPoolsField).(*schema.Set)

	if err := d.Set(AllocationPoolsField, schema.NewSet(allocationPoolsSet.F, allocationPoolsToListOfMaps(subnet.AllocationPools))); err != nil {
		return diag.FromErr(err)
	}

	d.Set(RegionIDField, subnet.RegionID)
	d.Set(ProjectIDField, subnet.ProjectID)
	d.Set(GatewayIPField, subnet.GatewayIP.String())

	fields := []string{ConnectToNetworkRouterField}
	revertState(d, &fields)

	if subnet.GatewayIP == nil {
		d.Set(ConnectToNetworkRouterField, false)
		d.Set(GatewayIPField, disable)
	}

	metadataMap, metadataReadOnly := PrepareMetadata(subnet.Metadata)

	if err = d.Set(MetadataMapField, metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set(MetadataReadOnlyField, metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish subnet reading")

	return diags
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet updating")
	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := &edgecloudV2.SubnetworkUpdateRequest{}

	if d.HasChange(NameField) {
		updateOpts.Name = d.Get(NameField).(string)
	}
	updateOpts.EnableDHCP = d.Get(EnableDHCPField).(bool)

	// In the structure, the field is mandatory for the ability to transfer the absence of data,
	// if you do not initialize it with a empty list, marshalling will send null and receive a validation error.
	dnsNameservers := d.Get(DNSNameserversField).([]interface{})
	updateOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		updateOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get(HostRoutesField).(*schema.Set).List()
	updateOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		updateOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	rawAPs, ok := d.GetOk(AllocationPoolsField)
	if ok {
		updateOpts.AllocationPools = prepareSubnetAllocationPools(rawAPs.([]interface{}))
	}

	switch {
	case d.HasChange(GatewayIPField):
		_, newValue := d.GetChange(GatewayIPField)
		if nV := newValue.(string); nV != disable && nV != "" {
			gatewayIP := net.ParseIP(newValue.(string))
			updateOpts.GatewayIP = &gatewayIP
		}
	default:
		if gIP := d.Get(GatewayIPField).(string); gIP != disable {
			gatewayIP := net.ParseIP(gIP)
			updateOpts.GatewayIP = &gatewayIP
		}
	}

	_, _, err = clientV2.Subnetworks.Update(ctx, subnetID, updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(MetadataMapField) {
		_, nmd := d.GetChange(MetadataMapField)
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

	d.Set(LastUpdatedField, time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish subnet updating")

	return resourceSubnetRead(ctx, d, m)
}

func resourceSubnetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)
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
