package reservedfixedip

import (
	"context"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/internal/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"time"
)

func ResourceEdgeCenterReservedFixedIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterReservedFixedIPCreate,
		ReadContext:   resourceEdgeCenterReservedFixedIPRead,
		UpdateContext: resourceEdgeCenterReservedFixedIPUpdate,
		DeleteContext: resourceEdgeCenterReservedFixedIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, ipID, err := utils.ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(ipID)

				return []*schema.ResourceData{d}, nil
			},
		},
		Description: `A reserved fixed IP is an IP address within a specific network that is reserved for a particular
purpose. Reserved fixed IPs are typically not automatically assigned to instances but are instead set aside for specific
needs or configurations`,
		Schema: reservedFixedIPSchema(),
	}
}

func resourceEdgeCenterReservedFixedIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP creating")
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.ReservedFixedIPCreateRequest{}

	if v, ok := d.GetOk("is_vip"); ok {
		opts.IsVIP = v.(bool)
	}

	newInstancePorts, ok := d.GetOk("instance_ports_that_share_vip")

	if ok && !opts.IsVIP {
		return diag.Errorf("field is_vip must be set 'true' for using field 'instance_ports_that_share_vip'")
	}

	portType := d.Get("type").(string)

	opts.Type = edgecloud.ReservedFixedIPType(portType)

	switch portType {
	case edgecloud.ReservedFixedIPTypeExternal:
	case edgecloud.ReservedFixedIPTypeSubnet:
		subnetID := d.Get("subnet_id").(string)
		if subnetID == "" {
			return diag.Errorf("'subnet_id' required if the type is 'subnet'")
		}

		opts.SubnetID = subnetID
	case edgecloud.ReservedFixedIPTypeAnySubnet:
		networkID := d.Get("network_id").(string)
		if networkID == "" {
			return diag.Errorf("'network_id' required if the type is 'any_subnet'")
		}
		opts.NetworkID = networkID
	case edgecloud.ReservedFixedIPTypeIPAddress:
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

	log.Printf("[DEBUG] Reserved fixed IP create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.ReservedFixedIP.Create, opts, client)
	if err != nil {
		return diag.Errorf("error creating reserved fixed IP: %s", err)
	}

	if ok && opts.IsVIP {
		newInstancePortsInterfaceSlice, ok := newInstancePorts.([]interface{})
		if !ok {
			return diag.Errorf("Error getting instance_ports_that_share_vip from api")
		}
		newInstancePortsStringSlice := make([]string, 0, len(newInstancePortsInterfaceSlice))
		for _, v := range newInstancePortsInterfaceSlice {
			vString, ok := v.(string)
			if !ok {
				return diag.Errorf("Error getting instance_ports_that_share_vip from api")
			}
			newInstancePortsStringSlice = append(newInstancePortsStringSlice, vString)
		}

		addInstancePortsRequest := edgecloud.AddInstancePortsRequest{PortIDs: newInstancePortsStringSlice}

		if _, _, err := client.ReservedFixedIP.AddInstancePorts(ctx, d.Id(), &addInstancePortsRequest); err != nil {
			return diag.Errorf("Error from replace instance ports: %s ", err)
		}
	}

	d.SetId(taskResult.Ports[0])

	log.Printf("[INFO] Reserved fixed IP: %s", d.Id())

	return resourceEdgeCenterReservedFixedIPRead(ctx, d, meta)
}

func resourceEdgeCenterReservedFixedIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the reserved fixed ip properties for updating the state
	reservedFixedIP, resp, err := client.ReservedFixedIP.Get(ctx, d.Id())
	if err != nil {
		// check if the reserved fixed ip no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] Removing reserved fixed ip %s because resource doesn't exist anymore", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving reserved fixed ip: %s", err)
	}
	d.Set("project_id", reservedFixedIP.ProjectID)
	d.Set("region_id", reservedFixedIP.RegionID)
	d.Set("status", reservedFixedIP.Status)
	d.Set("fixed_ip_address", reservedFixedIP.FixedIPAddress.String())
	d.Set("subnet_id", reservedFixedIP.SubnetID)
	d.Set("network_id", reservedFixedIP.NetworkID)
	d.Set("is_vip", reservedFixedIP.IsVIP)
	d.Set("port_id", reservedFixedIP.PortID)
	d.Set("name", reservedFixedIP.Name)
	d.Set("region_name", reservedFixedIP.Region)
	d.Set("is_external", reservedFixedIP.IsExternal)
	d.Set("network_name", reservedFixedIP.Network.Name)

	reservation := map[string]string{
		"status":        reservedFixedIP.Reservation.Status,
		"resource_type": reservedFixedIP.Reservation.ResourceType,
		"resource_id":   reservedFixedIP.Reservation.ResourceID,
	}
	d.Set("reservation", reservation)

	if reservedFixedIP.IsVIP {
		ports, _, err := client.ReservedFixedIP.ListInstancePorts(ctx, d.Id())
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
	allowedAddressPairs := make([]map[string]string, 0, len(reservedFixedIP.AllowedAddressPairs))
	for _, val := range reservedFixedIP.AllowedAddressPairs {
		allowedAddressPair := map[string]string{"ip_address": val.IPAddress, "mac_address": val.MacAddress}
		allowedAddressPairs = append(allowedAddressPairs, allowedAddressPair)
	}
	d.Set("allowed_address_pairs", allowedAddressPairs)

	log.Println("[DEBUG] Finish ReservedFixedIP reading")

	return nil
}

func resourceEdgeCenterReservedFixedIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP reading")
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	isVip := d.Get("is_vip").(bool)

	newInstancePorts, ok := d.GetOk("instance_ports_that_share_vip")

	if ok && !isVip {
		return diag.Errorf("field is_vip must be set 'true' for using field 'instance_ports_that_share_vip'")
	}

	if d.HasChange("is_vip") {
		newIsVIP := d.Get("is_vip")
		switchVIPRequest := edgecloud.SwitchVIPStatusRequest{IsVIP: newIsVIP.(bool)}
		_, _, err := client.ReservedFixedIP.SwitchVIPStatus(ctx, d.Id(), &switchVIPRequest)
		if err != nil {
			return diag.FromErr(err)
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	if d.HasChange("instance_ports_that_share_vip") && isVip {
		newInstancePortsInterfaceSlice, ok := newInstancePorts.([]interface{})
		if !ok {
			return diag.Errorf("Error getting instance_ports_that_share_vip from api")
		}
		newInstancePortsStringSlice := make([]string, 0, len(newInstancePortsInterfaceSlice))
		for _, v := range newInstancePortsInterfaceSlice {
			vString, ok := v.(string)
			if !ok {
				return diag.Errorf("Error getting instance_ports_that_share_vip from api")
			}
			newInstancePortsStringSlice = append(newInstancePortsStringSlice, vString)
		}

		addInstancePortsRequest := edgecloud.AddInstancePortsRequest{PortIDs: newInstancePortsStringSlice}
		if _, _, err := client.ReservedFixedIP.ReplaceInstancePorts(ctx, d.Id(), &addInstancePortsRequest); err != nil {
			return diag.Errorf("Error from replace instance ports: %s ", err)
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}
	log.Println("[DEBUG] Finish ReservedFixedIP updating")
	return resourceEdgeCenterReservedFixedIPRead(ctx, d, meta)
}

func resourceEdgeCenterReservedFixedIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ReservedFixedIP deleting")
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting reserved fixed ip: %s", d.Id())
	isVip := d.Get("is_vip").(bool)
	if isVip {
		if _, _, err := client.ReservedFixedIP.SwitchVIPStatus(ctx, d.Id(), &edgecloud.SwitchVIPStatusRequest{IsVIP: false}); err != nil {
			return diag.Errorf("Error switching is_vip status to false , before deleting: %s", err)
		}
	}

	task, _, err := client.ReservedFixedIP.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("Error deleting reserved fixed ip: %s", err)
	}
	if err = util.WaitForTaskComplete(ctx, client, task.Tasks[0]); err != nil {
		return diag.Errorf("Delete reserved fixed ip task failed with error: %s", err)
	}

	if err = util.ResourceIsDeleted(ctx, client.ReservedFixedIP.Get, d.Id()); err != nil {
		return diag.Errorf("reserved fixed ip with id %s was not deleted: %s", d.Id(), err)
	}
	d.SetId("")
	log.Printf("[DEBUG] Finish of ReservedFixedIP deleting")
	return nil
}
