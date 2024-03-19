package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	ReservedFixedIPsPoint        = "reserved_fixed_ips"
	ReservedFixedIPCreateTimeout = 1200 * time.Second
	ReservedFixedIPDeleteTimeout = 1200 * time.Second
)

func resourceReservedFixedIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceReservedFixedIPCreate,
		ReadContext:   resourceReservedFixedIPRead,
		UpdateContext: resourceReservedFixedIPUpdate,
		DeleteContext: resourceReservedFixedIPDelete,
		Description:   "Represent reserved ips",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, ipID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(ipID)

				return []*schema.ResourceData{d}, nil
			},
		},

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
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: fmt.Sprintf("The type of reserved fixed IP. Valid values are '%s', '%s', '%s', and '%s'", edgecloudV2.ReservedFixedIPTypeExternal, edgecloudV2.ReservedFixedIPTypeSubnet, edgecloudV2.ReservedFixedIPTypeAnySubnet, edgecloudV2.ReservedFixedIPTypeIPAddress),
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					switch edgecloudV2.ReservedFixedIPType(v) {
					case edgecloudV2.ReservedFixedIPTypeExternal, edgecloudV2.ReservedFixedIPTypeSubnet, edgecloudV2.ReservedFixedIPTypeAnySubnet, edgecloudV2.ReservedFixedIPTypeIPAddress:
						return diag.Diagnostics{}
					}
					return diag.Errorf("wrong type %s, available values is '%s', '%s', '%s', '%s'", v, edgecloudV2.ReservedFixedIPTypeExternal, edgecloudV2.ReservedFixedIPTypeSubnet, edgecloudV2.ReservedFixedIPTypeAnySubnet, edgecloudV2.ReservedFixedIPTypeIPAddress)
				},
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the reserved fixed IP.",
			},
			"fixed_ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The IP address that is associated with the reserved IP.",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
			},
			"subnet_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "ID of the subnet from which the fixed IP should be reserved.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "ID of the network to which the reserved fixed IP is associated.",
			},
			"is_vip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag to determine if the reserved fixed IP should be treated as a Virtual IP (VIP).",
			},
			"port_id": {
				Type:        schema.TypeString,
				Description: "ID of the port_id underlying the reserved fixed IP.",
				Computed:    true,
			},
			"allowed_address_pairs": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Group of IP addresses that share the current IP as VIP.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Optional: true,
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
			"reservation": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The status of the reserved fixed IP with the type of the resource and the ID it is attached to",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"instance_ports_that_share_vip": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "instance ports that share a VIP",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceReservedFixedIPCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	opts := &edgecloudV2.ReservedFixedIPCreateRequest{}

	if v, ok := d.GetOk("is_vip"); ok {
		opts.IsVIP = v.(bool)
	}

	newInstancePorts, okInstancePorts := d.GetOk("instance_ports_that_share_vip")

	if okInstancePorts && !opts.IsVIP {
		return diag.Errorf("field is_vip must be set 'true' for using field 'instance_ports_that_share_vip'")
	}

	allowedAddressPairsRaw, okAllowedAddressPairs := d.GetOk("allowed_address_pairs")
	var allowedAddressPairs []interface{}
	if okAllowedAddressPairs {
		if opts.IsVIP {
			return diag.Errorf("You can't use allowed_address_pairs with VIP IPs")
		}
		allowedAddressPairs = allowedAddressPairsRaw.([]interface{})
	}

	portType := d.Get("type").(string)
	switch edgecloudV2.ReservedFixedIPType(portType) {
	case edgecloudV2.ReservedFixedIPTypeExternal:
	case edgecloudV2.ReservedFixedIPTypeSubnet:
		subnetID := d.Get("subnet_id").(string)
		if subnetID == "" {
			return diag.Errorf("'subnet_id' required if the type is 'subnet'")
		}

		opts.SubnetID = subnetID
	case edgecloudV2.ReservedFixedIPTypeAnySubnet:
		networkID := d.Get("network_id").(string)
		if networkID == "" {
			return diag.Errorf("'network_id' required if the type is 'any_subnet'")
		}
		opts.NetworkID = networkID
	case edgecloudV2.ReservedFixedIPTypeIPAddress:
		networkID := d.Get("network_id").(string)
		ipAddress := d.Get("fixed_ip_address").(string)
		if networkID == "" || ipAddress == "" {
			return diag.Errorf("'network_id' and 'fixed_ip_address' required if the type is 'ip_address'")
		}

		opts.NetworkID = networkID
		opts.IPAddress = ipAddress
	default:
		return diag.Errorf("wrong type %s, available values is 'external', 'subnet', 'any_subnet', 'ip_address'", portType)
	}

	opts.Type = edgecloudV2.ReservedFixedIPType(portType)

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.ReservedFixedIP.Create, opts, clientV2, ReservedFixedIPCreateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}
	reservedFixedIPID := taskResult.Ports[0]

	d.SetId(reservedFixedIPID)

	if okInstancePorts && opts.IsVIP {
		newInstancePortsInterfaceSlice := newInstancePorts.(*schema.Set).List()
		newInstancePortsStringSlice := make([]string, 0, len(newInstancePortsInterfaceSlice))
		for _, v := range newInstancePortsInterfaceSlice {
			vString, _ := v.(string)
			newInstancePortsStringSlice = append(newInstancePortsStringSlice, vString)
		}

		addInstancePortsRequest := edgecloudV2.AddInstancePortsRequest{PortIDs: newInstancePortsStringSlice}
		// Retry attempts to add instance ports to VIP (adding instance ports to VIP requires attaching interface to instance, this operation may take some time)
		err = retryAddInstancePorts(ctx, clientV2, reservedFixedIPID, addInstancePortsRequest)
		if err != nil {
			return diag.Errorf("Error from adding instance ports to VIP: %s ", err)
		}
	}

	if okAllowedAddressPairs && !opts.IsVIP {
		// Retry attempts to assign allowed address pairs (because assigning address pairs requires attaching interface to instance, this operation may take some time)
		err = retryAllowedAddressPairs(ctx, clientV2, reservedFixedIPID, allowedAddressPairs)
		if err != nil {
			return diag.Errorf("error from assigning AllowedAddressPairs: %s", err)
		}
	}

	log.Printf("[DEBUG] ReservedFixedIP id (%s)", reservedFixedIPID)

	diags = append(diags, resourceReservedFixedIPRead(ctx, d, m)...)

	log.Printf("[DEBUG] Finish ReservedFixedIP creating (%s)", reservedFixedIPID)

	return diags
}

func resourceReservedFixedIPRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	reservedFixedIP, resp, err := clientV2.ReservedFixedIP.Get(ctx, d.Id())
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Removing reserved fixed ip %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project_id", reservedFixedIP.ProjectID)
	d.Set("region_id", reservedFixedIP.RegionID)
	d.Set("status", reservedFixedIP.Status)
	d.Set("fixed_ip_address", reservedFixedIP.FixedIPAddress.String())
	d.Set("subnet_id", reservedFixedIP.SubnetID)
	d.Set("network_id", reservedFixedIP.NetworkID)
	d.Set("is_vip", reservedFixedIP.IsVIP)
	d.Set("port_id", reservedFixedIP.PortID)
	// d.Set("last_updated", reservedFixedIP.UpdatedAt)

	reservation := map[string]string{
		"status":        reservedFixedIP.Reservation.Status,
		"resource_type": reservedFixedIP.Reservation.ResourceType,
		"resource_id":   reservedFixedIP.Reservation.ResourceID,
	}
	d.Set("reservation", reservation)

	if reservedFixedIP.IsVIP {
		ports, _, err := clientV2.ReservedFixedIP.ListInstancePorts(ctx, d.Id())
		instancePorts := make([]string, 0, len(ports))
		if err != nil {
			return diag.Errorf("Error from getting instance ports that share a VIP: %s", err)
		}
		if len(ports) != 0 {
			for _, port := range ports {
				instancePorts = append(instancePorts, port.PortID)
			}
		}
		if err = d.Set("instance_ports_that_share_vip", instancePorts); err != nil {
			return diag.FromErr(err)
		}
	}

	allowedPairs := make([]map[string]interface{}, len(reservedFixedIP.AllowedAddressPairs))
	for i, p := range reservedFixedIP.AllowedAddressPairs {
		pair := make(map[string]interface{})

		pair["ip_address"] = p.IPAddress
		pair["mac_address"] = p.MacAddress

		allowedPairs[i] = pair
	}
	if err := d.Set("allowed_address_pairs", allowedPairs); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish ReservedFixedIP reading")

	return diags
}

func resourceReservedFixedIPUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP updating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	_, isVipNew := d.GetChange("is_vip")
	isVip := isVipNew.(bool)

	oldInstancePorts, newInstancePorts := d.GetChange("instance_ports_that_share_vip")
	newInstancePortsSet := newInstancePorts.(*schema.Set)
	oldInstancePortsSet := oldInstancePorts.(*schema.Set)

	var instancePortsThatShareVip []interface{}
	if newInstancePortsSet.Len() != 0 {
		instancePortsThatShareVip = newInstancePortsSet.List()
	}
	if len(instancePortsThatShareVip) != 0 && !isVip {
		return diag.Errorf("field is_vip must be set 'true' for using field 'instance_ports_that_share_vip'")
	}

	id := d.Id()
	if d.HasChange("is_vip") {
		opts := &edgecloudV2.SwitchVIPStatusRequest{IsVIP: d.Get("is_vip").(bool)}
		_, _, err := clientV2.ReservedFixedIP.SwitchVIPStatus(ctx, id, opts)
		if err != nil {
			return diag.FromErr(err)
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("allowed_address_pairs") {
		_, newAllowedAddressPairs := d.GetChange("allowed_address_pairs")
		allowedAddressPairs := newAllowedAddressPairs.([]interface{})
		if assignDiag := assignAllowedAddressPairs(ctx, clientV2, id, allowedAddressPairs); assignDiag != nil {
			// Retry attempts to assign allowed address pairs (because assigning address pairs requires attaching interface to instance, this operation may take some time)
			err = retryAllowedAddressPairs(ctx, clientV2, id, allowedAddressPairs)
			if err != nil {
				return diag.Errorf("error from assigning AllowedAddressPairs: %s", err)
			}
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("instance_ports_that_share_vip") && isVip {
		var isReplaceInstancePorts bool
		switch {
		case oldInstancePortsSet.Len() != 0 && newInstancePortsSet.Len() != 0:
			portsIntersection := newInstancePortsSet.Intersection(oldInstancePortsSet)
			if portsIntersection.Len() != oldInstancePortsSet.Len() {
				isReplaceInstancePorts = true
			}
		case newInstancePortsSet.Len() == 0:
			isReplaceInstancePorts = true
		}

		newInstancePortsStringSlice := make([]string, 0, len(instancePortsThatShareVip))
		for _, v := range instancePortsThatShareVip {
			vString, ok := v.(string)
			if !ok {
				return diag.Errorf("Error getting instance_ports_that_share_vip from api")
			}
			newInstancePortsStringSlice = append(newInstancePortsStringSlice, vString)
		}

		addInstancePortsRequest := edgecloudV2.AddInstancePortsRequest{PortIDs: newInstancePortsStringSlice}

		switch isReplaceInstancePorts {
		case false:
			// Retry attempts to add instance ports to VIP (adding instance ports to VIP requires attaching interface to instance, this operation may take some time)
			err = retryAddInstancePorts(ctx, clientV2, id, addInstancePortsRequest)
		case true:
			// Retry attempts to replace instance ports to VIP (adding instance ports to VIP requires attaching interface to instance, this operation may take some time)
			err = retryReplaceInstancePorts(ctx, clientV2, id, addInstancePortsRequest)
		}
		if err != nil {
			return diag.Errorf("Error from replace instance ports: %s ", err)
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	log.Println("[DEBUG] Finish ReservedFixedIP updating")

	diags = append(diags, resourceReservedFixedIPRead(ctx, d, m)...)

	return diags
}

func resourceReservedFixedIPDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	isVip := d.Get("is_vip").(bool)
	if isVip {
		if _, _, err := clientV2.ReservedFixedIP.SwitchVIPStatus(ctx, d.Id(), &edgecloudV2.SwitchVIPStatusRequest{IsVIP: false}); err != nil {
			return diag.Errorf("Error switching is_vip status to false , before deleting: %s", err)
		}
	}

	id := d.Id()
	results, resp, err := clientV2.ReservedFixedIP.Delete(ctx, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of ReservedFixedIP deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, ReservedFixedIPDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of ReservedFixedIP deleting")

	return diags
}
