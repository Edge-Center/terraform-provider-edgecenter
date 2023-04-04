package edgecenter

import (
	"crypto/md5"
	"encoding/binary"
	"io"
	"strconv"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	typesSG "github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/types"
)

const (
	minPort = 0
	maxPort = 65535
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

// validatePortRange checks if the provided port value is within the valid range.
func validatePortRange(v interface{}, _ cty.Path) diag.Diagnostics {
	val := v.(int)
	if val >= minPort && val <= maxPort {
		return nil
	}
	return diag.Errorf("available range %d-%d", minPort, maxPort)
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
