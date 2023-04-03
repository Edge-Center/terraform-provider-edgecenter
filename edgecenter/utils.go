package edgecenter

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"

	dnsSDK "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter"
)

const (
	VersionPointV1 = "v1"
	VersionPointV2 = "v2"

	ProjectPoint = "projects"
	RegionPoint  = "regions"
)

type Config struct {
	Provider      *edgecloud.ProviderClient
	CDNClient     cdn.ClientService
	StorageClient *storageSDK.SDK
	DNSClient     *dnsSDK.Client
}

// MapStructureDecoder decodes the given map into the provided structure using the specified decoder configuration.
func MapStructureDecoder(strct interface{}, v *map[string]interface{}, config *mapstructure.DecoderConfig) error {
	config.Result = strct
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	return decoder.Decode(*v)
}

// ImportStringParser parses a string containing project ID, region ID, and another field,
// and returns them as separate values along with any error encountered.
func ImportStringParser(infoStr string) (int, int, string, error) {
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 3 {
		return 0, 0, "", fmt.Errorf("failed import: wrong input id: %s", infoStr)
	}
	projectID, err := strconv.Atoi(infoStrings[0])
	if err != nil {
		return 0, 0, "", err
	}
	regionID, err := strconv.Atoi(infoStrings[1])
	if err != nil {
		return 0, 0, "", err
	}

	return projectID, regionID, infoStrings[2], nil
}

// CreateClient creates a new edgecloud.ServiceClient.
func CreateClient(provider *edgecloud.ProviderClient, d *schema.ResourceData, endpoint string, version string) (*edgecloud.ServiceClient, error) {
	projectID, err := GetProject(provider, d.Get("project_id").(int), d.Get("project_name").(string))
	if err != nil {
		return nil, err
	}

	regionID := 0

	rawRegionID := d.Get("region_id")
	rawRegionName := d.Get("region_name")
	if rawRegionID != nil && rawRegionName != nil {
		regionID, err = GetRegion(provider, rawRegionID.(int), rawRegionName.(string))
		if err != nil {
			return nil, fmt.Errorf("failed to get region: %w", err)
		}
	}

	client, err := edgecenter.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    endpoint,
		Region:  regionID,
		Project: projectID,
		Version: version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// revertState reverts the state of the specified fields in the given schema.ResourceData if "last_updated" is not empty.
// It takes a schema.ResourceData and a slice of strings containing the field names to be reverted as input arguments.
func revertState(d *schema.ResourceData, fields *[]string) {
	if d.Get("last_updated").(string) != "" {
		for _, field := range *fields {
			if d.HasChange(field) {
				oldValue, _ := d.GetChange(field)
				switch v := oldValue.(type) {
				case int:
					d.Set(field, v)
				case string:
					d.Set(field, v)
				case map[string]interface{}:
					d.Set(field, v)
				}
			}
			log.Printf("[DEBUG] Revert (%s) '%s' field", d.Id(), field)
		}
	}
}

// ExtractHostAndPath splits a given URI into the host and path components.
func ExtractHostAndPath(uri string) (string, string, error) {
	var host, path string
	if uri == "" {
		return host, path, fmt.Errorf("empty uri")
	}

	pURL, err := url.Parse(uri)
	if err != nil {
		return host, path, fmt.Errorf("url parse: %w", err)
	}
	host = pURL.Scheme + "://" + pURL.Host
	path = pURL.Path

	return host, path, nil
}
