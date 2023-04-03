package edgecenter

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/lbpools"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/listeners"
	typesLb "github.com/Edge-Center/edgecentercloud-go/edgecenter/loadbalancer/v1/types"
)

// ImportStringParserExtended parses a string containing project ID, region ID, and two other fields,
// and returns them as separate values along with any error encountered.
func ImportStringParserExtended(infoStr string) (int, int, string, string, error) {
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 4 {
		return 0, 0, "", "", fmt.Errorf("failed import: wrong input id: %s", infoStr)
	}
	projectID, err := strconv.Atoi(infoStrings[0])
	if err != nil {
		return 0, 0, "", "", err
	}
	regionID, err := strconv.Atoi(infoStrings[1])
	if err != nil {
		return 0, 0, "", "", err
	}

	return projectID, regionID, infoStrings[2], infoStrings[3], nil
}

// extractSessionPersistenceMap creates a session persistence options struct from the data in the given ResourceData.
func extractSessionPersistenceMap(d *schema.ResourceData) *lbpools.CreateSessionPersistenceOpts {
	var sessionOpts *lbpools.CreateSessionPersistenceOpts
	sessionPersistence := d.Get("session_persistence").([]interface{})
	if len(sessionPersistence) > 0 {
		sm := sessionPersistence[0].(map[string]interface{})
		sessionOpts = &lbpools.CreateSessionPersistenceOpts{
			Type: typesLb.PersistenceType(sm["type"].(string)),
		}

		granularity, ok := sm["persistence_granularity"].(string)
		if ok {
			sessionOpts.PersistenceGranularity = granularity
		}

		timeout, ok := sm["persistence_timeout"].(int)
		if ok {
			sessionOpts.PersistenceTimeout = timeout
		}

		cookieName, ok := sm["cookie_name"].(string)
		if ok {
			sessionOpts.CookieName = cookieName
		}
	}

	return sessionOpts
}

// extractHealthMonitorMap creates a health monitor options struct from the data in the given ResourceData.
func extractHealthMonitorMap(d *schema.ResourceData) *lbpools.CreateHealthMonitorOpts {
	var healthOpts *lbpools.CreateHealthMonitorOpts
	monitors := d.Get("health_monitor").([]interface{})
	if len(monitors) > 0 {
		hm := monitors[0].(map[string]interface{})
		healthOpts = &lbpools.CreateHealthMonitorOpts{
			Type:       typesLb.HealthMonitorType(hm["type"].(string)),
			Delay:      hm["delay"].(int),
			MaxRetries: hm["max_retries"].(int),
			Timeout:    hm["timeout"].(int),
		}

		maxRetriesDown := hm["max_retries_down"].(int)
		if maxRetriesDown != 0 {
			healthOpts.MaxRetriesDown = maxRetriesDown
		}

		httpMethod := hm["http_method"].(string)
		if httpMethod != "" {
			healthOpts.HTTPMethod = typesLb.HTTPMethodPointer(typesLb.HTTPMethod(httpMethod))
		}

		urlPath := hm["url_path"].(string)
		if urlPath != "" {
			healthOpts.URLPath = urlPath
		}

		expectedCodes := hm["expected_codes"].(string)
		if expectedCodes != "" {
			healthOpts.ExpectedCodes = expectedCodes
		}

		id := hm["id"].(string)
		if id != "" {
			healthOpts.ID = id
		}
	}

	return healthOpts
}

// extractListenerIntoMap converts a listener object into a map.
func extractListenerIntoMap(listener *listeners.Listener) map[string]interface{} {
	l := make(map[string]interface{})
	l["id"] = listener.ID
	l["name"] = listener.Name
	l["protocol"] = listener.Protocol.String()
	l["protocol_port"] = listener.ProtocolPort
	l["secret_id"] = listener.SecretID
	l["sni_secret_id"] = listener.SNISecretID
	return l
}
