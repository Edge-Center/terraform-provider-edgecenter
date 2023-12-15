package converter

import (
	"net"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func ListInterfaceToListInstanceVolumeCreate(volumes []interface{}) ([]edgecloud.InstanceVolumeCreate, error) {
	vols := make([]edgecloud.InstanceVolumeCreate, len(volumes))
	for i, volume := range volumes {
		vol := volume.(map[string]interface{})
		var V edgecloud.InstanceVolumeCreate
		if err := MapStructureDecoder(&V, &vol, decoderConfig); err != nil {
			return nil, err
		}
		vols[i] = V
	}

	return vols, nil
}

func ListInterfaceToListInstanceInterface(interfaces []interface{}) ([]edgecloud.InstanceInterface, error) {
	ifs := make([]edgecloud.InstanceInterface, len(interfaces))
	for idx, i := range interfaces {
		inter := i.(map[string]interface{})
		I := edgecloud.InstanceInterface{
			Type:      edgecloud.InterfaceType(inter["type"].(string)),
			NetworkID: inter["network_id"].(string),
			PortID:    inter["port_id"].(string),
			SubnetID:  inter["subnet_id"].(string),
		}

		switch inter["floating_ip_source"].(string) {
		case "new":
			I.FloatingIP = &edgecloud.InterfaceFloatingIP{
				Source: edgecloud.NewFloatingIP,
			}
		case "existing":
			I.FloatingIP = &edgecloud.InterfaceFloatingIP{
				Source:             edgecloud.ExistingFloatingIP,
				ExistingFloatingID: inter["floating_ip"].(string),
			}
		default:
			I.FloatingIP = nil
		}

		sgList := inter["security_groups"].([]interface{})
		if len(sgList) > 0 {
			sgs := make([]edgecloud.ID, 0, len(sgList))
			for _, sg := range sgList {
				sgs = append(sgs, edgecloud.ID{ID: sg.(string)})
			}
			I.SecurityGroups = sgs
		} else {
			I.SecurityGroups = []edgecloud.ID{}
		}

		ifs[idx] = I
	}

	return ifs, nil
}

// ListInterfaceToLoadbalancerSessionPersistence creates a session persistence options struct.
func ListInterfaceToLoadbalancerSessionPersistence(sessionPersistence []interface{}) *edgecloud.LoadbalancerSessionPersistence {
	var sp *edgecloud.LoadbalancerSessionPersistence

	if len(sessionPersistence) > 0 {
		sessionPersistenceMap := sessionPersistence[0].(map[string]interface{})
		sp = &edgecloud.LoadbalancerSessionPersistence{
			Type: edgecloud.SessionPersistence(sessionPersistenceMap["type"].(string)),
		}

		if granularity, ok := sessionPersistenceMap["persistence_granularity"].(string); ok {
			sp.PersistenceGranularity = granularity
		}

		if timeout, ok := sessionPersistenceMap["persistence_timeout"].(int); ok {
			sp.PersistenceTimeout = timeout
		}

		if cookieName, ok := sessionPersistenceMap["cookie_name"].(string); ok {
			sp.CookieName = cookieName
		}
	}

	return sp
}

// ListInterfaceToHealthMonitor creates a heath monitor options struct.
func ListInterfaceToHealthMonitor(healthMonitor []interface{}) edgecloud.HealthMonitorCreateRequest {
	var hm edgecloud.HealthMonitorCreateRequest

	if len(healthMonitor) > 0 {
		healthMonitorMap := healthMonitor[0].(map[string]interface{})
		hm = edgecloud.HealthMonitorCreateRequest{
			Timeout:    healthMonitorMap["timeout"].(int),
			Delay:      healthMonitorMap["delay"].(int),
			Type:       edgecloud.HealthMonitorType(healthMonitorMap["type"].(string)),
			MaxRetries: healthMonitorMap["max_retries"].(int),
		}

		if httpMethod, ok := healthMonitorMap["http_method"].(string); ok {
			hm.HTTPMethod = edgecloud.HTTPMethod(httpMethod)
		}

		if urlPath, ok := healthMonitorMap["url_path"].(string); ok {
			hm.URLPath = urlPath
		}

		if maxRetriesDown, ok := healthMonitorMap["max_retries_down"].(int); ok {
			hm.MaxRetriesDown = maxRetriesDown
		}

		if expectedCodes, ok := healthMonitorMap["expected_codes"].(string); ok {
			hm.ExpectedCodes = expectedCodes
		}
	}

	return hm
}

func ListInterfaceToListPoolMember(poolMembers []interface{}) ([]edgecloud.PoolMemberCreateRequest, error) {
	members := make([]edgecloud.PoolMemberCreateRequest, len(poolMembers))
	for i, member := range poolMembers {
		m := member.(map[string]interface{})
		address := m["address"].(string)
		m["address"] = net.ParseIP(address)
		var M edgecloud.PoolMemberCreateRequest
		if err := MapStructureDecoder(&M, &m, decoderConfig); err != nil {
			return nil, err
		}
		members[i] = M
	}

	return members, nil
}
