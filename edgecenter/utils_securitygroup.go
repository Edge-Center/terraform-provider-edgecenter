package edgecenter

import (
	"crypto/md5"
	"encoding/binary"
	"io"
	"strconv"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	typesSG "github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/types"
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

// extractSecurityGroupRuleMap creates a security group rule from the provided map and security group ID.
func extractSecurityGroupRuleMap(r interface{}, gid string) securitygroups.CreateSecurityGroupRuleOpts {
	rule := r.(map[string]interface{})

	opts := securitygroups.CreateSecurityGroupRuleOpts{
		Direction:       typesSG.RuleDirection(rule["direction"].(string)),
		EtherType:       typesSG.EtherType(rule["ethertype"].(string)),
		Protocol:        typesSG.Protocol(rule["protocol"].(string)),
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
