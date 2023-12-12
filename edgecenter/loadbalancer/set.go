package loadbalancer

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func setFlavor(_ context.Context, d *schema.ResourceData, loadbalancer *edgecloud.Loadbalancer) error {
	flavor := map[string]interface{}{
		"flavor_name": loadbalancer.Flavor.FlavorName,
		"vcpus":       strconv.Itoa(loadbalancer.Flavor.VCPUS),
		"ram":         strconv.Itoa(loadbalancer.Flavor.RAM),
		"flavor_id":   loadbalancer.Flavor.FlavorID,
	}

	return d.Set("flavor", flavor)
}

func setMetadataDetailed(_ context.Context, d *schema.ResourceData, loadbalancer *edgecloud.Loadbalancer) error {
	if len(loadbalancer.MetadataDetailed) > 0 {
		metadata := make([]map[string]interface{}, 0, len(loadbalancer.MetadataDetailed))
		for _, metadataItem := range loadbalancer.MetadataDetailed {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}

		return d.Set("metadata_detailed", metadata)
	}

	return nil
}

func setVRRPIPs(_ context.Context, d *schema.ResourceData, loadbalancer *edgecloud.Loadbalancer) error {
	if len(loadbalancer.VrrpIPs) > 0 {
		vrrpIPs := make([]string, 0, len(loadbalancer.VrrpIPs))
		for _, v := range loadbalancer.VrrpIPs {
			vrrpIPs = append(vrrpIPs, v.VrrpIPAddress)
		}
		return d.Set("vrrp_ips", vrrpIPs)
	}

	return nil
}

func setFloatingIP(_ context.Context, d *schema.ResourceData, loadbalancer *edgecloud.Loadbalancer) error {
	if len(loadbalancer.FloatingIPs) > 0 {
		floatingIP := loadbalancer.FloatingIPs[0]
		fip := map[string]interface{}{
			"status":              floatingIP.Status,
			"id":                  floatingIP.ID,
			"fixed_ip_address":    floatingIP.FixedIPAddress.String(),
			"floating_ip_address": floatingIP.FloatingIPAddress,
			"router_id":           floatingIP.RouterID,
			"port_id":             floatingIP.PortID,
		}

		return d.Set("floating_ip", fip)
	}

	return nil
}
