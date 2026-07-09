package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// resolveListenerAfterLBMigration searches for a listener across all project listeners
// by primary fields (name, protocol, protocol_port) and narrows with secondary fields
// when multiple candidates exist. Always confirms with secondary fields on single match.
func resolveListenerAfterLBMigration(
	ctx context.Context, clientV2 *edgecloudV2.Client,
	name string, protocol string, protocolPort int,
	allowedCIDRs []string,
	timeoutClientData, timeoutMemberData, timeoutMemberConnect *int,
	secretID string, sniSecretIDs []string,
) (*edgecloudV2.Listener, error) {
	listeners, _, err := clientV2.Loadbalancers.ListenerList(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list listeners for migration rebind: %w", err)
	}

	var matches []edgecloudV2.Listener
	for _, l := range listeners {
		if l.Name == name && string(l.Protocol) == protocol && l.ProtocolPort == protocolPort {
			matches = append(matches, l)
		}
	}

	if len(matches) == 0 {
		//nolint: nilnil
		return nil, nil
	}

	// filter by secondary fields
	var exact []edgecloudV2.Listener
	for _, l := range matches {
		if listenerSecondaryMatch(l, allowedCIDRs, timeoutClientData, timeoutMemberData, timeoutMemberConnect, secretID, sniSecretIDs) {
			exact = append(exact, l)
		}
	}

	switch {
	case len(exact) == 1:
		return &exact[0], nil
	case len(matches) == 1:
		log.Printf("[WARN] listener rebind: single primary match for %s/%s/%d but secondary fields differ; accepting primary match",
			name, protocol, protocolPort)
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous listener rebind: %d listeners match name=%s protocol=%s port=%d (secondary narrowed to %d)",
			len(matches), name, protocol, protocolPort, len(exact))
	}
}

func listenerSecondaryMatch(l edgecloudV2.Listener,
	allowedCIDRs []string,
	timeoutClientData, timeoutMemberData, timeoutMemberConnect *int,
	secretID string, sniSecretIDs []string,
) bool {
	if len(l.AllowedCIDRs) != len(allowedCIDRs) {
		return false
	}
	for i := range l.AllowedCIDRs {
		if l.AllowedCIDRs[i] != allowedCIDRs[i] {
			return false
		}
	}
	if ptrOrZero(l.TimeoutClientData) != ptrOrZero(timeoutClientData) {
		return false
	}
	if ptrOrZero(l.TimeoutMemberData) != ptrOrZero(timeoutMemberData) {
		return false
	}
	if ptrOrZero(l.TimeoutMemberConnect) != ptrOrZero(timeoutMemberConnect) {
		return false
	}
	if l.SecretID != secretID {
		return false
	}
	if len(l.SNISecretID) != len(sniSecretIDs) {
		return false
	}
	for i := range l.SNISecretID {
		if l.SNISecretID[i] != sniSecretIDs[i] {
			return false
		}
	}

	return true
}

func ptrOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// resolvePoolAfterLBMigration searches for a pool across all project pools
// by primary fields (name, protocol, lb_algorithm) and narrows with secondary fields.
func resolvePoolAfterLBMigration(
	ctx context.Context, clientV2 *edgecloudV2.Client,
	name string, protocol string, lbAlgorithm string,
	healthMonitorType string, healthMonitorDelay, healthMonitorTimeout, healthMonitorMaxRetries int,
	healthMonitorMaxRetriesDown int, healthMonitorURLPath, healthMonitorExpectedCodes string,
	sessionPersistenceType, sessionPersistenceCookieName string,
) (*edgecloudV2.Pool, error) {
	allPools, _, err := clientV2.Loadbalancers.PoolList(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pools for migration rebind: %w", err)
	}

	var matches []edgecloudV2.Pool
	for _, p := range allPools {
		if p.Name == name && string(p.Protocol) == protocol && string(p.LoadbalancerAlgorithm) == lbAlgorithm {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		//nolint: nilnil
		return nil, nil
	}

	var exact []edgecloudV2.Pool
	for _, p := range matches {
		if poolSecondaryMatch(p, healthMonitorType, healthMonitorDelay, healthMonitorTimeout, healthMonitorMaxRetries,
			healthMonitorMaxRetriesDown, healthMonitorURLPath, healthMonitorExpectedCodes,
			sessionPersistenceType, sessionPersistenceCookieName) {
			exact = append(exact, p)
		}
	}

	switch {
	case len(exact) == 1:
		return &exact[0], nil
	case len(matches) == 1:
		log.Printf("[WARN] pool rebind: single primary match for %s/%s/%s but secondary fields differ; accepting primary match",
			name, protocol, lbAlgorithm)
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous pool rebind: %d pools match name=%s protocol=%s algorithm=%s (secondary narrowed to %d)",
			len(matches), name, protocol, lbAlgorithm, len(exact))
	}
}

func poolSecondaryMatch(p edgecloudV2.Pool,
	healthMonitorType string, healthMonitorDelay, healthMonitorTimeout, healthMonitorMaxRetries int,
	healthMonitorMaxRetriesDown int, healthMonitorURLPath, healthMonitorExpectedCodes string,
	sessionPersistenceType, sessionPersistenceCookieName string,
) bool {
	if p.HealthMonitor != nil {
		if string(p.HealthMonitor.Type) != healthMonitorType {
			return false
		}
		if p.HealthMonitor.Delay != healthMonitorDelay {
			return false
		}
		if p.HealthMonitor.Timeout != healthMonitorTimeout {
			return false
		}
		if p.HealthMonitor.MaxRetries != healthMonitorMaxRetries {
			return false
		}
		if p.HealthMonitor.MaxRetriesDown != healthMonitorMaxRetriesDown {
			return false
		}
		if p.HealthMonitor.URLPath != healthMonitorURLPath {
			return false
		}
		if p.HealthMonitor.ExpectedCodes != healthMonitorExpectedCodes {
			return false
		}
	} else if healthMonitorType != "" {
		return false
	}

	if p.SessionPersistence != nil {
		if string(p.SessionPersistence.Type) != sessionPersistenceType {
			return false
		}
		if p.SessionPersistence.CookieName != sessionPersistenceCookieName {
			return false
		}
	} else if sessionPersistenceType != "" {
		return false
	}

	return true
}

// resolveMemberAcrossPools searches for a member by address+protocol_port across all pools.
// Uses weight, subnetID, instanceID as secondary discriminators.
func resolveMemberAcrossPools(
	ctx context.Context, clientV2 *edgecloudV2.Client,
	address net.IP, protocolPort int,
	weight int, subnetID string, instanceID string,
) (*edgecloudV2.PoolMember, string, error) {
	allPools, _, err := clientV2.Loadbalancers.PoolList(ctx, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list pools for member migration rebind: %w", err)
	}

	type poolMemberPair struct {
		member edgecloudV2.PoolMember
		poolID string
	}

	var primary []poolMemberPair
	for _, p := range allPools {
		for _, m := range p.Members {
			if m.Address.Equal(address) && m.ProtocolPort == protocolPort {
				primary = append(primary, poolMemberPair{member: m, poolID: p.ID})
			}
		}
	}

	if len(primary) == 0 {
		return nil, "", nil
	}

	// secondary: prefer exact weight+subnet+instance match
	var exact []poolMemberPair
	for _, pr := range primary {
		if pr.member.Weight == weight && pr.member.SubnetID == subnetID && pr.member.InstanceID == instanceID {
			exact = append(exact, pr)
		}
	}

	switch {
	case len(exact) == 1:
		return &exact[0].member, exact[0].poolID, nil
	case len(primary) == 1:
		log.Printf("[WARN] member rebind: single primary match for %s/%d but secondary fields differ; accepting primary match",
			address.String(), protocolPort)
		return &primary[0].member, primary[0].poolID, nil
	default:
		return nil, "", fmt.Errorf("ambiguous member rebind: %d members match address=%s port=%d across pools (secondary narrowed to %d)",
			len(primary), address.String(), protocolPort, len(exact))
	}
}

// resolveL7PolicyAfterLBMigration searches for an L7 policy across the project
// by primary fields (name, action) and narrows with secondary fields.
func resolveL7PolicyAfterLBMigration(
	ctx context.Context, clientV2 *edgecloudV2.Client,
	name string, action string,
	position int,
	redirectPoolID, redirectURL, redirectPrefix string,
	redirectHTTPCode int,
) (*edgecloudV2.L7Policy, error) {
	policies, _, err := clientV2.L7Policies.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list l7 policies for migration rebind: %w", err)
	}

	var matches []edgecloudV2.L7Policy
	for _, p := range policies {
		if p.Name == name && string(p.Action) == action {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		// retry with action only (name may be empty)
		for _, p := range policies {
			if string(p.Action) == action {
				matches = append(matches, p)
			}
		}
	}

	if len(matches) == 0 {
		//nolint: nilnil
		return nil, nil
	}

	var exact []edgecloudV2.L7Policy
	for _, p := range matches {
		if l7PolicySecondaryMatch(p, position, redirectPoolID, redirectURL, redirectPrefix, redirectHTTPCode) {
			exact = append(exact, p)
		}
	}

	switch {
	case len(exact) == 1:
		return &exact[0], nil
	case len(matches) == 1:
		log.Printf("[WARN] l7 policy rebind: single primary match for name=%s action=%s but secondary fields differ; accepting primary match",
			name, action)
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous l7 policy rebind: %d policies match name=%s action=%s (secondary narrowed to %d)",
			len(matches), name, action, len(exact))
	}
}

func l7PolicySecondaryMatch(p edgecloudV2.L7Policy,
	position int,
	redirectPoolID, redirectURL, redirectPrefix string,
	redirectHTTPCode int,
) bool {
	if p.Position != position {
		return false
	}

	poolID := ""
	if p.RedirectPoolID != nil {
		poolID = *p.RedirectPoolID
	}
	if poolID != redirectPoolID {
		return false
	}

	url := ""
	if p.RedirectURL != nil {
		url = *p.RedirectURL
	}
	if url != redirectURL {
		return false
	}

	prefix := ""
	if p.RedirectPrefix != nil {
		prefix = *p.RedirectPrefix
	}
	if prefix != redirectPrefix {
		return false
	}

	code := 0
	if p.RedirectHTTPCode != nil {
		code = *p.RedirectHTTPCode
	}
	if code != redirectHTTPCode {
		return false
	}

	return true
}

// resolveL7RuleAfterPolicyMigration searches for an L7 rule within a known policy
// by all identifying attributes.
func resolveL7RuleAfterPolicyMigration(ctx context.Context, clientV2 *edgecloudV2.Client,
	l7policyID string,
	ruleType string, key string, value string, compareType string, invert bool,
) (*edgecloudV2.L7Rule, error) {
	rules, _, err := clientV2.L7Rules.List(ctx, l7policyID)
	if err != nil {
		return nil, err
	}

	var matches []edgecloudV2.L7Rule
	for _, r := range rules {
		if string(r.Type) == ruleType && r.Key == key && r.Value == value && string(r.CompareType) == compareType && r.Invert == invert {
			matches = append(matches, r)
		}
	}

	if len(matches) == 0 {
		//nolint: nilnil
		return nil, nil
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous l7 rule rebind: %d rules match type=%s key=%s value=%s compareType=%s invert=%v in policy %s",
			len(matches), ruleType, key, value, compareType, invert, l7policyID)
	}

	return &matches[0], nil
}
