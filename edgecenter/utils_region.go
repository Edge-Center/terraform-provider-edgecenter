package edgecenter

import (
	"fmt"
	"log"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/region/v1/regions"
)

// findRegionByName searches for a region with the specified name in the provided region slice.
// Returns the region ID if found, otherwise returns an error.
func findRegionByName(arr []regions.Region, name string) (int, error) {
	for _, el := range arr {
		if el.DisplayName == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("region with name %s not found", name)
}

// GetRegion returns a valid region ID for a resource.
// If the regionID is provided, it will be returned directly.
// If regionName is provided instead, the function will search for the region by name and return its ID.
// Returns an error if the region is not found or there is an issue with the client.
func GetRegion(provider *edgecloud.ProviderClient, regionID int, regionName string) (int, error) {
	if regionID != 0 {
		return regionID, nil
	}
	client, err := edgecenter.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    RegionPoint,
		Region:  0,
		Project: 0,
		Version: VersionPointV1,
	})
	if err != nil {
		return 0, err
	}

	rs, err := regions.ListAll(client)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] Regions: %v", rs)
	regionID, err = findRegionByName(rs, regionName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the region is successful: regionID=%d", regionID)

	return regionID, nil
}
