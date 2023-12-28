package utils

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
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

// ParseImportString parses a string containing project ID, region ID, and another field,
// and returns them as separate values along with any error encountered.
func ParseImportString(infoStr string) (projectID int, regionID int, id3 string, err error) { // nolint: nonamedreturns
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

// RevertState reverts the state of the specified fields in the given schema.ResourceData if "last_updated" is not empty.
// It takes a schema.ResourceData and a slice of strings containing the field names to be reverted as input arguments.
func RevertState(d *schema.ResourceData, fields *[]string) {
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
