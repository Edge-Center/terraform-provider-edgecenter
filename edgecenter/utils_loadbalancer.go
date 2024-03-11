package edgecenter

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// ImportStringParserExtended parses a string containing project ID, region ID, and two other fields,
// and returns them as separate values along with any error encountered.
func ImportStringParserExtended(infoStr string) (projectID int, regionID int, id3 string, id4 string, err error) { // nolint: nonamedreturns
	log.Printf("[DEBUG] Input id string: %s", infoStr)
	infoStrings := strings.Split(infoStr, ":")
	if len(infoStrings) != 4 {
		err = fmt.Errorf("failed import: wrong input id: %s", infoStr)
		return
	}

	id1, id2, id3, id4 := infoStrings[0], infoStrings[1], infoStrings[2], infoStrings[3]

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

// extractSessionPersistenceMapV2 creates a session persistence options struct from the data in the given ResourceData.
func extractSessionPersistenceMapV2(d *schema.ResourceData) *edgecloudV2.LoadbalancerSessionPersistence {
	var sessionOpts *edgecloudV2.LoadbalancerSessionPersistence
	sessionPersistence := d.Get("session_persistence").([]interface{})
	if len(sessionPersistence) > 0 {
		sm := sessionPersistence[0].(map[string]interface{})
		sessionOpts = &edgecloudV2.LoadbalancerSessionPersistence{
			Type: edgecloudV2.SessionPersistence(sm["type"].(string)),
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

// extractHealthMonitorMapV2 creates a health monitor options struct from the data in the given ResourceData.
func extractHealthMonitorMapV2(d *schema.ResourceData) *edgecloudV2.HealthMonitorCreateRequest {
	var healthOpts *edgecloudV2.HealthMonitorCreateRequest
	monitors := d.Get("health_monitor").([]interface{})
	if len(monitors) > 0 {
		hm := monitors[0].(map[string]interface{})
		healthOpts = &edgecloudV2.HealthMonitorCreateRequest{
			Type:       edgecloudV2.HealthMonitorType(hm["type"].(string)),
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
			hm := edgecloudV2.HTTPMethod(httpMethod)
			healthOpts.HTTPMethod = &hm
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

// extractListenerIntoMapV2 converts a listener object into a map.
func extractListenerIntoMapV2(listener *edgecloudV2.Listener) map[string]interface{} {
	l := make(map[string]interface{})
	l["id"] = listener.ID
	l["name"] = listener.Name
	l["protocol"] = listener.Protocol
	l["protocol_port"] = listener.ProtocolPort
	l["secret_id"] = listener.SecretID
	l["sni_secret_id"] = listener.SNISecretID
	return l
}
