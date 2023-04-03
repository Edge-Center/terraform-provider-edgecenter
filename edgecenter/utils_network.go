package edgecenter

import (
	"encoding/json"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/availablenetworks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
)

// findNetworkByName searches for a network with the given name among the given networks.
// Returns the found network and a flag indicating the success of the search.
func findNetworkByName(name string, nets []networks.Network) (networks.Network, bool) {
	for _, n := range nets {
		if n.Name == name {
			return n, true
		}
	}
	return networks.Network{}, false
}

// findSharedNetworkByName searches for a shared network with the given name among the given networks.
// Returns the found network and a flag indicating the success of the search.
func findSharedNetworkByName(name string, nets []availablenetworks.Network) (availablenetworks.Network, bool) {
	for _, n := range nets {
		if n.Name == name {
			return n, true
		}
	}
	return availablenetworks.Network{}, false
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
