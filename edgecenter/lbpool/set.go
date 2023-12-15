package lbpool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func setHealthMonitor(_ context.Context, d *schema.ResourceData, pool *edgecloud.Pool) error {
	healthMonitor := []map[string]interface{}{{
		"id":               pool.HealthMonitor.ID,
		"type":             pool.HealthMonitor.Type,
		"delay":            pool.HealthMonitor.Delay,
		"timeout":          pool.HealthMonitor.Timeout,
		"max_retries":      pool.HealthMonitor.MaxRetries,
		"max_retries_down": pool.HealthMonitor.MaxRetriesDown,
		"url_path":         pool.HealthMonitor.URLPath,
		"expected_codes":   pool.HealthMonitor.ExpectedCodes,
	}}

	return d.Set("healthmonitor", healthMonitor)
}

func setSessionPersistence(_ context.Context, d *schema.ResourceData, pool *edgecloud.Pool) error {
	sessionPersistence := []map[string]interface{}{{
		"type":                    pool.SessionPersistence.Type,
		"cookie_name":             pool.SessionPersistence.CookieName,
		"persistence_timeout":     pool.SessionPersistence.PersistenceTimeout,
		"persistence_granularity": pool.SessionPersistence.PersistenceGranularity,
	}}

	return d.Set("session_persistence", sessionPersistence)
}

func setMembers(_ context.Context, d *schema.ResourceData, pool *edgecloud.Pool) error {
	members := make([]map[string]interface{}, 0, len(pool.Members))
	for _, m := range pool.Members {
		member := map[string]interface{}{
			"id":               m.ID,
			"operating_status": m.OperatingStatus,
			"weight":           m.Weight,
			"address":          m.Address.String(),
			"protocol_port":    m.ProtocolPort,
			"subnet_id":        m.SubnetID,
			"instance_id":      m.InstanceID,
			"admin_state_up":   m.AdminStateUP,
		}
		members = append(members, member)
	}

	return d.Set("member", members)
}
