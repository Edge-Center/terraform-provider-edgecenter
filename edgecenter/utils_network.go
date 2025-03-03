package edgecenter

import (
	"encoding/json"
	"net"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// findNetworkByName searches for a network with the given name among the given networks.
// Returns the found network and a flag indicating the success of the search.
func findNetworkByName(name string, nets []edgecloudV2.Network) (edgecloudV2.Network, bool) {
	for _, n := range nets {
		if n.Name == name {
			return n, true
		}
	}
	return edgecloudV2.Network{}, false
}

// findSharedNetworkByName searches for a shared network with the given name among the given networks.
// Returns the found network and a flag indicating the success of the search.
func findSharedNetworkByName(name string, nets []edgecloudV2.NetworkSubnetwork) (edgecloudV2.NetworkSubnetwork, bool) {
	for _, n := range nets {
		if n.Name == name {
			return n, true
		}
	}
	return edgecloudV2.NetworkSubnetwork{}, false
}

// StructToMap converts the struct to map[string]interface{}.
// Returns an error if the conversion fails.
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var newMap map[string]interface{}
	err = json.Unmarshal(data, &newMap)
	if err != nil {
		return nil, err
	}

	return newMap, nil
}

func dnsNameserversToStringList(dnsNameservers []net.IP) []string {
	dns := make([]string, len(dnsNameservers))
	for i, ns := range dnsNameservers {
		dns[i] = ns.String()
	}

	return dns
}

func hostRoutesToListOfMapsV2(hostRoutes []edgecloudV2.HostRoute) []map[string]string {
	hrs := make([]map[string]string, len(hostRoutes))
	for i, hr := range hostRoutes {
		hR := map[string]string{"destination": "", "nexthop": ""}
		hR["destination"] = hr.Destination.String()
		hR["nexthop"] = hr.NextHop.String()
		hrs[i] = hR
	}

	return hrs
}

func prepareSubnets(subs []edgecloudV2.Subnetwork) []map[string]interface{} {
	subnetList := make([]map[string]interface{}, 0, len(subs))
	for _, s := range subs {
		subnetList = append(subnetList, map[string]interface{}{
			"id":              s.ID,
			"name":            s.Name,
			"enable_dhcp":     s.EnableDHCP,
			"cidr":            s.CIDR,
			"available_ips":   s.AvailableIps,
			"total_ips":       s.TotalIps,
			"has_router":      s.HasRouter,
			"dns_nameservers": dnsNameserversToStringList(s.DNSNameservers),
			"host_routes":     hostRoutesToListOfMapsV2(s.HostRoutes),
			"gateway_ip":      s.GatewayIP.String(),
		})
	}

	return subnetList
}

func prepareResellerNetwork(rn edgecloudV2.ResellerNetwork) map[string]interface{} {
	network := make(map[string]interface{})

	network[CreatedAtField] = rn.CreatedAt
	network[DefaultField] = rn.Default
	network[ExternalField] = rn.External
	network[SharedField] = rn.Shared
	network[IDField] = rn.ID
	network[MTUField] = rn.MTU
	network[NameField] = rn.Name
	network[RegionIDField] = rn.RegionID
	network[RegionNameField] = rn.Region
	network[TypeField] = rn.Type
	network[SubnetsField] = prepareSubnets(rn.Subnets)
	network[CreatorTaskIDField] = rn.CreatorTaskID
	network[TaskIDField] = rn.TaskID
	network[SegmentationIDField] = rn.SegmentationID
	network[UpdatedAtField] = rn.UpdatedAt
	network[ClientIDField] = rn.ClientID
	network[ProjectIDField] = rn.RegionID
	network[MetadataField] = PrepareMetadataReadonly(rn.Metadata)

	return network
}

func resellerNetworksCloudClientConf() *CloudClientConf {
	return &CloudClientConf{
		DoNotUseRegionID:  true,
		DoNotUseProjectID: true,
	}
}
