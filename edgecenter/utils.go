package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/region/v1/regions"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	VersionPointV1 = "v1"
	VersionPointV2 = "v2"

	ProjectPoint = "projects"
	RegionPoint  = "regions"
)

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
func ImportStringParser(infoStr string) (projectID int, regionID int, id3 string, err error) { //nolint:nonamedreturns
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 3 {
		err = fmt.Errorf("failed import: wrong input id: %s", infoStr)
		return
	}

	id1, id2, id3 := infoStrings[0], infoStrings[1], infoStrings[2]

	projectID, err = strconv.Atoi(id1)
	if err != nil {
		return
	}
	regionID, err = strconv.Atoi(id2)
	if err != nil {
		return
	}

	return
}

// findRegionByNameLegacy to support backwards compatibility.
func findRegionByNameLegacy(arr []regions.Region, name string) (int, error) {
	for _, el := range arr {
		if el.DisplayName == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("region with name %s not found", name)
}

// GetRegionLegacy to support backwards compatibility.
func GetRegionLegacy(provider *edgecloud.ProviderClient, regionID int, regionName string) (int, error) {
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
	regionID, err = findRegionByNameLegacy(rs, regionName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the region is successful: regionID=%d", regionID)

	return regionID, nil
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
		regionID, err = GetRegionLegacy(provider, rawRegionID.(int), rawRegionName.(string))
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

// GetRegionIDandProjectID search for project ID and region ID by name or return project ID
// and region ID if they exist in the terraform configuration.
// Use new version Edgecenterclient-go V2.
// nolint: nonamedreturns
func GetRegionIDandProjectID(
	ctx context.Context,
	client *edgecloudV2.Client,
	d *schema.ResourceData,
) (regionID int, projectID int, err error) {
	regionID, err = GetRegionID(ctx, client, d)
	if err != nil {
		return 0, 0, err
	}
	projectID, err = GetProjectID(ctx, client, d)
	if err != nil {
		return 0, 0, err
	}

	return regionID, projectID, nil
}

func GetRegionID(
	ctx context.Context,
	client *edgecloudV2.Client,
	d *schema.ResourceData,
) (int, error) {
	rID, IDOk := d.GetOk("region_id")
	rName, NameOk := d.GetOk("region_name")

	if !IDOk && !NameOk {
		return 0, fmt.Errorf("both parameters and region_id and region_name are not provided")
	}

	regionID, err := GetRegionV2(ctx, client, rID.(int), rName.(string))
	if err != nil {
		return 0, fmt.Errorf("failed to get region: %w", err)
	}

	return regionID, nil
}

func GetProjectID(
	ctx context.Context,
	client *edgecloudV2.Client,
	d *schema.ResourceData,
) (int, error) {
	pID, IDOk := d.GetOk("project_id")
	pName, NameOk := d.GetOk("project_name")

	if !IDOk && !NameOk {
		return 0, fmt.Errorf("both parameters and project_id and project_name are not provided")
	}

	project, err := GetProjectV2(ctx, client, strconv.Itoa(pID.(int)), pName.(string))
	if err != nil {
		return 0, err
	}

	return project.ID, nil
}

func validateURLFunc(v interface{}, attributeName string) (warnings []string, errors []error) { //nolint:nonamedreturns
	value, ok := v.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %s to be string", attributeName))
		return
	}

	_, err := url.ParseRequestURI(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("URL is not valid: %s", err.Error()))
	}

	return warnings, errors
}

// IndexFunc returns the first index i satisfying f(s[i]),
// or -1 if none do.
// TODO remove when upgrading to a new version - https://tracker.yandex.ru/CLOUDDEV-456.
func IndexFunc[E any](s []E, f func(E) bool) int {
	for i := range s {
		if f(s[i]) {
			return i
		}
	}
	return -1
}
