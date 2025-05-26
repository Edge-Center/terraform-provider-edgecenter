package edgecenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type rawNetworkType = map[string]interface{}

type fetchNetworksWithSubnetsOptions struct {
	clientV2    *edgecloudV2.Client
	fetchOpts   *edgecloudV2.NetworksWithSubnetsOptions
	networkName string
}

// findNetworkByName searches for a network with the given name among the given networks.
// Returns the found network and a flag indicating the success of the search.
func findNetworkByName(name string, nets []edgecloudV2.Network) (edgecloudV2.Network, error) {
	var foundNets []edgecloudV2.Network
	for _, n := range nets {
		if n.Name == name {
			foundNets = append(foundNets, n)
		}
	}

	switch {
	case len(foundNets) == 0:
		return edgecloudV2.Network{}, fmt.Errorf("network with name: %s does not exist", name)

	case len(foundNets) > 1:
		var message bytes.Buffer
		message.WriteString("Found networks:\n")

		for _, ntw := range foundNets {
			message.WriteString(fmt.Sprintf("  - ID: %s\n", ntw.ID))
		}

		return edgecloudV2.Network{}, fmt.Errorf("multiple networks exist. %s\nUse network ID instead of name", message.String())
	}

	return foundNets[0], nil
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

func fetchNetworksWithSubnets(ctx context.Context, opts fetchNetworksWithSubnetsOptions) (rawNetworkType, []edgecloudV2.Subnetwork, []edgecloudV2.MetadataDetailed, error) {
	nets, _, err := opts.clientV2.Networks.ListNetworksWithSubnets(ctx, opts.fetchOpts)
	if err != nil {
		return nil, nil, nil, err
	}

	foundNets := make([]edgecloudV2.NetworkSubnetwork, 0, len(nets))

	if opts.fetchOpts == nil {
		for _, ntw := range nets {
			if ntw.Name == opts.networkName {
				foundNets = append(foundNets, ntw)
			}
		}
	}

	switch {
	case len(foundNets) == 0:
		return nil, nil, nil, errors.New("network does not exist")

	case len(foundNets) > 1:
		var message bytes.Buffer
		message.WriteString("Found networks:\n")

		for _, fnet := range foundNets {
			message.WriteString(fmt.Sprintf("  - ID: %s\n", fnet.ID))
		}

		return nil, nil, nil, fmt.Errorf("multiple networks found.\n %s\n Use network ID instead of name", message.String())
	}

	ntw := foundNets[0]

	rawNetwork, err := StructToMap(ntw)
	if err != nil {
		return nil, nil, nil, err
	}

	return rawNetwork, ntw.Subnets, ntw.Metadata, nil
}
