package edgecenter

import (
	"crypto/md5"
	"encoding/binary"
	"io"
	"strconv"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// secGroupUniqueID generates a unique ID for a security group rule using its properties.
func secGroupUniqueID(i interface{}) int {
	e := i.(map[string]interface{})

	h := md5.New()
	proto, _ := e["protocol"].(string)
	io.WriteString(h, e["direction"].(string))
	io.WriteString(h, e["ethertype"].(string))
	io.WriteString(h, proto)
	io.WriteString(h, strconv.Itoa(e["port_range_min"].(int)))
	io.WriteString(h, strconv.Itoa(e["port_range_max"].(int)))
	io.WriteString(h, e["description"].(string))
	io.WriteString(h, e["remote_ip_prefix"].(string))

	return int(binary.BigEndian.Uint64(h.Sum(nil)))
}

// extractSecurityGroupRuleCreateRequestV2 creates a security group rule from the provided map and security group ID.
func extractSecurityGroupRuleCreateRequestV2(r interface{}, gid string) edgecloudV2.RuleCreateRequest {
	rule := r.(map[string]interface{})

	opts := edgecloudV2.RuleCreateRequest{
		Direction:       edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)),
		EtherType:       edgecloudV2.EtherType(rule["ethertype"].(string)),
		Protocol:        edgecloudV2.SecurityGroupRuleProtocol(rule["protocol"].(string)),
		SecurityGroupID: &gid,
	}

	minP, maxP := rule["port_range_min"].(int), rule["port_range_max"].(int)
	if minP != 0 && maxP != 0 {
		opts.PortRangeMin = &minP
		opts.PortRangeMax = &maxP
	}

	description, _ := rule["description"].(string)
	opts.Description = &description

	remoteIPPrefix := rule["remote_ip_prefix"].(string)
	if remoteIPPrefix != "" {
		opts.RemoteIPPrefix = &remoteIPPrefix
	}

	return opts
}

// extractSecurityGroupRuleUpdateRequestV2 creates a security group rule from the provided map and security group ID.
func extractSecurityGroupRuleUpdateRequestV2(r interface{}, gid string) edgecloudV2.RuleUpdateRequest {
	rule := r.(map[string]interface{})

	opts := edgecloudV2.RuleUpdateRequest{
		Direction:       edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)),
		EtherType:       edgecloudV2.EtherType(rule["ethertype"].(string)),
		Protocol:        edgecloudV2.SecurityGroupRuleProtocol(rule["protocol"].(string)),
		SecurityGroupID: gid,
	}

	minP, maxP := rule["port_range_min"].(int), rule["port_range_max"].(int)
	if minP != 0 && maxP != 0 {
		opts.PortRangeMin = minP
		opts.PortRangeMax = maxP
	}

	description, _ := rule["description"].(string)
	opts.Description = description

	remoteIPPrefix := rule["remote_ip_prefix"].(string)
	if remoteIPPrefix != "" {
		opts.RemoteIPPrefix = remoteIPPrefix
	}

	return opts
}
