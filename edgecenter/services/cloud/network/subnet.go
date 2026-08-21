package network

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/shared/tfutil"
)

const (
	SubnetCreatingTimeout = 1200 * time.Second
	SubnetDeleteTimeout   = 1200 * time.Second
	SubnetPoint           = "subnets"
	disable               = "disable"
	SubnetResource        = "edgecenter_subnet"
)

var (
	errSubnetDeleteLocked  = errors.New("subnet delete is locked")
	errSubnetDeletePending = errors.New("subnet delete is still in progress")
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
				projectID, regionID, subnetID, err := edgecenter.ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set(edgecenter.ProjectIDField, projectID)
				d.Set(edgecenter.RegionIDField, regionID)
				d.SetId(subnetID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			edgecenter.ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
			},
			edgecenter.RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{edgecenter.RegionIDField, edgecenter.RegionNameField},
			},
			edgecenter.NameField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the subnet.",
			},
			edgecenter.EnableDHCPField: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.",
			},
			edgecenter.CIDRField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Represents the IP address range of the subnet.",
			},
			edgecenter.NetworkIDField: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the network to which this subnet belongs.",
			},
			edgecenter.ConnectToNetworkRouterField: {
				Type:        schema.TypeBool,
				Description: "True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null. Default true.",
				Optional:    true,
				Default:     true,
			},
			edgecenter.DNSNameserversField: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of DNS name servers for the subnet.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			edgecenter.HostRoutesField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Set of additional routes to be added to instances that are part of this subnet.",
				Elem:        HostRouteSchema(true),
			},
			edgecenter.GatewayIPField: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The IP address of the gateway for this subnet. The subnet will be recreated if the gateway IP is changed.",
				ValidateFunc:     validateSubnetGatewayIP,
				DiffSuppressFunc: suppressSubnetGatewayIPDiff,
			},
			edgecenter.AllocationPoolsField: {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: "A list of allocation pools for DHCP. If omitted but DHCP or gateway settings are changed on update, pools are automatically reassigned.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						edgecenter.StartField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Start IP address.",
							ValidateFunc: validation.IsIPAddress,
						},
						edgecenter.EndField: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "End IP address.",
							ValidateFunc: validation.IsIPAddress,
						},
					},
				},
			},
			edgecenter.MetadataMapField: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
			edgecenter.LastUpdatedField: {
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	createOpts := &edgecloudV2.SubnetworkCreateRequest{
		Name:                   d.Get(edgecenter.NameField).(string),
		EnableDHCP:             d.Get(edgecenter.EnableDHCPField).(bool),
		NetworkID:              d.Get(edgecenter.NetworkIDField).(string),
		ConnectToNetworkRouter: d.Get(edgecenter.ConnectToNetworkRouterField).(bool),
	}

	rawAPs, ok := d.GetOk(edgecenter.AllocationPoolsField)
	if ok {
		createOpts.AllocationPools = prepareSubnetAllocationPools(rawAPs.(*schema.Set).List())
	}

	cidr := d.Get(edgecenter.CIDRField).(string)
	if cidr != "" {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return diag.FromErr(err)
		}
		createOpts.CIDR = cidr
	}

	dnsNameservers := d.Get(edgecenter.DNSNameserversField).([]interface{})
	createOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		createOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get(edgecenter.HostRoutesField).(*schema.Set).List()
	createOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		createOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	gatewayIP := d.Get(edgecenter.GatewayIPField).(string)
	gw := net.ParseIP(gatewayIP)
	if gatewayIP == disable {
		createOpts.ConnectToNetworkRouter = false
	} else if gw != nil {
		createOpts.GatewayIP = &gw
	}

	if metadataRaw, ok := d.GetOk(edgecenter.MetadataMapField); ok {
		meta, err := edgecenter.MapInterfaceToMapString(metadataRaw)
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

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)
	subnet, _, err := clientV2.Subnetworks.Get(ctx, subnetID)
	if err != nil {
		return diag.Errorf("cannot get subnet with ID: %s. Error: %s", subnetID, err)
	}

	d.Set(edgecenter.NameField, subnet.Name)
	d.Set(edgecenter.EnableDHCPField, subnet.EnableDHCP)
	d.Set(edgecenter.CIDRField, subnet.CIDR)
	d.Set(edgecenter.NetworkIDField, subnet.NetworkID)

	dns := make([]string, len(subnet.DNSNameservers))
	for i, ns := range subnet.DNSNameservers {
		dns[i] = ns.String()
	}
	d.Set(edgecenter.DNSNameserversField, dns)

	hrs := make([]map[string]string, len(subnet.HostRoutes))
	for i, hr := range subnet.HostRoutes {
		hR := map[string]string{edgecenter.DestinationField: "", edgecenter.NexthopField: ""}
		hR[edgecenter.DestinationField] = hr.Destination.String()
		hR[edgecenter.NexthopField] = hr.NextHop.String()
		hrs[i] = hR
	}

	d.Set(edgecenter.HostRoutesField, hrs)

	allocationPoolsSet := d.Get(edgecenter.AllocationPoolsField).(*schema.Set)

	if err := d.Set(edgecenter.AllocationPoolsField, schema.NewSet(allocationPoolsSet.F, allocationPoolsToListOfMaps(subnet.AllocationPools))); err != nil {
		return diag.FromErr(err)
	}

	d.Set(edgecenter.RegionIDField, subnet.RegionID)
	d.Set(edgecenter.ProjectIDField, subnet.ProjectID)
	fields := []string{edgecenter.ConnectToNetworkRouterField}
	tfutil.RevertState(d, &fields)
	if subnet.GatewayIP != nil {
		d.Set(edgecenter.GatewayIPField, subnet.GatewayIP.String())
	} else {
		d.Set(edgecenter.GatewayIPField, disable)
		d.Set(edgecenter.ConnectToNetworkRouterField, false)
	}

	metadataMap, metadataReadOnly := edgecenter.PrepareMetadata(subnet.Metadata)

	if err = d.Set(edgecenter.MetadataMapField, metadataMap); err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set(edgecenter.MetadataReadOnlyField, metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish subnet reading")

	return diags
}

func resourceSubnetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet updating")
	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := &edgecloudV2.SubnetworkUpdateRequest{}

	if d.HasChange(edgecenter.NameField) {
		updateOpts.Name = d.Get(edgecenter.NameField).(string)
	}
	updateOpts.EnableDHCP = d.Get(edgecenter.EnableDHCPField).(bool)

	// In the structure, the field is mandatory for the ability to transfer the absence of data,
	// if you do not initialize it with a empty list, marshalling will send null and receive a validation error.
	dnsNameservers := d.Get(edgecenter.DNSNameserversField).([]interface{})
	updateOpts.DNSNameservers = make([]net.IP, 0)
	if len(dnsNameservers) > 0 {
		ns := dnsNameservers
		dns := make([]net.IP, len(ns))
		for i, s := range ns {
			dns[i] = net.ParseIP(s.(string))
		}
		updateOpts.DNSNameservers = dns
	}

	hostRoutes := d.Get(edgecenter.HostRoutesField).(*schema.Set).List()
	updateOpts.HostRoutes = make([]edgecloudV2.HostRoute, 0)
	if len(hostRoutes) > 0 {
		updateOpts.HostRoutes, err = extractHostRoutesMapV2(hostRoutes)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	rawAPs, ok := d.GetOk(edgecenter.AllocationPoolsField)
	if ok {
		rawAPsList := rawAPs.(*schema.Set).List()
		updateOpts.AllocationPools = prepareSubnetAllocationPools(rawAPsList)
	}

	switch {
	case d.HasChange(edgecenter.GatewayIPField):
		_, newValue := d.GetChange(edgecenter.GatewayIPField)
		if nV := newValue.(string); nV != disable && nV != "" {
			gatewayIP := net.ParseIP(newValue.(string))
			updateOpts.GatewayIP = &gatewayIP
		}
	default:
		if gIP := d.Get(edgecenter.GatewayIPField).(string); gIP != disable && gIP != "" {
			gatewayIP := net.ParseIP(gIP)
			updateOpts.GatewayIP = &gatewayIP
		}
	}

	_, _, err = clientV2.Subnetworks.Update(ctx, subnetID, updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(edgecenter.MetadataMapField) {
		_, nmd := d.GetChange(edgecenter.MetadataMapField)
		meta, err := edgecenter.MapInterfaceToMapString(nmd)
		if err != nil {
			return diag.Errorf("metadata wrong fmt. Error: %s", err)
		}

		metaSubnet := edgecloudV2.Metadata(*meta)

		_, err = clientV2.Subnetworks.MetadataUpdate(ctx, subnetID, &metaSubnet)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	d.Set(edgecenter.LastUpdatedField, time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish subnet updating")

	return resourceSubnetRead(ctx, d, m)
}

func resourceSubnetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start subnet deleting")
	var diags diag.Diagnostics

	clientV2, err := edgecenter.InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	subnetID := d.Id()
	log.Printf("[DEBUG] Subnet id = %s", subnetID)

	err = deleteSubnetWithRetry(ctx, clientV2, subnetID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of subnet deleting")

	return diags
}

func deleteSubnetWithRetry(ctx context.Context, clientV2 *edgecloudV2.Client, subnetID string) error {
	ctx, cancel := context.WithTimeout(ctx, SubnetDeleteTimeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastErr error

	for {
		err := deleteSubnetOnce(ctx, clientV2, subnetID)
		if err == nil {
			return nil
		}

		if !errors.Is(err, errSubnetDeleteLocked) && !errors.Is(err, errSubnetDeletePending) {
			return err
		}

		lastErr = err
		log.Printf("[DEBUG] Retrying subnet delete for %s: %v", subnetID, err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout deleting subnet %s after retryable error: %w", subnetID, lastErr)
		case <-ticker.C:
		}
	}
}

func deleteSubnetOnce(ctx context.Context, clientV2 *edgecloudV2.Client, subnetID string) error {
	results, resp, err := clientV2.Subnetworks.Delete(ctx, subnetID)
	if err != nil {
		switch {
		case utilV2.IsNotFoundErr(resp):
			return nil
		case utilV2.IsLockedErr(resp):
			return fmt.Errorf("%w: %w", errSubnetDeleteLocked, err)
		default:
			return fmt.Errorf("delete subnet %s: %w", subnetID, err)
		}
	}

	if results == nil || len(results.Tasks) == 0 {
		return ensureSubnetDeleted(ctx, clientV2, subnetID)
	}

	_, err = utilV2.WaitAndGetTaskInfo(ctx, clientV2, results.Tasks[0], SubnetDeleteTimeout)
	if err != nil {
		checkErr := ensureSubnetDeleted(ctx, clientV2, subnetID)
		switch {
		case checkErr == nil:
			return nil
		case errors.Is(checkErr, errSubnetDeletePending):
			return fmt.Errorf("%w: subnet delete task failed but subnet still exists: %w", errSubnetDeletePending, err)
		default:
			return checkErr
		}
	}

	return ensureSubnetDeleted(ctx, clientV2, subnetID)
}

func ensureSubnetDeleted(ctx context.Context, clientV2 *edgecloudV2.Client, subnetID string) error {
	exists, err := utilV2.ResourceIsExist(ctx, clientV2.Subnetworks.Get, subnetID)
	switch {
	case err != nil:
		return fmt.Errorf("check subnet %s existence after delete: %w", subnetID, err)
	case !exists:
		return nil
	default:
		return fmt.Errorf("%w: subnet %s still exists after delete attempt", errSubnetDeletePending, subnetID)
	}
}

func suppressSubnetGatewayIPDiff(_, oldValue, newValue string, d *schema.ResourceData) bool {
	if newValue != "" {
		return false
	}
	connectToRouter := d.Get(edgecenter.ConnectToNetworkRouterField).(bool)
	if oldValue != "" && oldValue != disable && connectToRouter {
		return true
	}
	if oldValue == disable && !connectToRouter {
		return true
	}

	return false
}
