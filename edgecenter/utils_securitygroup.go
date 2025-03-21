package edgecenter

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type (
	portRangeMax = int
	portRangeMin = int
)

var networkProtocolWithPort = map[edgecloudV2.SecurityGroupRuleProtocol]struct{}{
	edgecloudV2.SGRuleProtocolTCP:     {},
	edgecloudV2.SGRuleProtocolUDP:     {},
	edgecloudV2.SGRuleProtocolUDPLITE: {},
	edgecloudV2.SGRuleProtocolSCTP:    {},
	edgecloudV2.SGRuleProtocolDCCP:    {},
}

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
func extractSecurityGroupRuleCreateRequestV2(r interface{}, gid string) (edgecloudV2.RuleCreateRequest, error) {
	var err error
	rule := r.(map[string]interface{})

	protocol := edgecloudV2.SecurityGroupRuleProtocol(rule["protocol"].(string))

	opts := edgecloudV2.RuleCreateRequest{
		Direction:       edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)),
		EtherType:       edgecloudV2.EtherType(rule["ethertype"].(string)),
		Protocol:        protocol,
		SecurityGroupID: &gid,
	}

	opts.PortRangeMin, opts.PortRangeMax, err = validatePortRange(protocol, rule)
	if err != nil {
		return edgecloudV2.RuleCreateRequest{}, err
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

	return opts, nil
}

// extractSecurityGroupRuleUpdateRequestV2 creates a security group rule from the provided map and security group ID.
func extractSecurityGroupRuleUpdateRequestV2(r interface{}, gid string) (edgecloudV2.RuleUpdateRequest, error) {
	rule := r.(map[string]interface{})

	protocol := edgecloudV2.SecurityGroupRuleProtocol(rule["protocol"].(string))

	opts := edgecloudV2.RuleUpdateRequest{
		Direction:       edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)),
		EtherType:       edgecloudV2.EtherType(rule["ethertype"].(string)),
		Protocol:        protocol,
		SecurityGroupID: gid,
	}

	portRangeMin, portRangeMax, err := validatePortRange(protocol, rule)
	if err != nil {
		return edgecloudV2.RuleUpdateRequest{}, err
	}

	opts.PortRangeMin, opts.PortRangeMax = *portRangeMin, *portRangeMax

	description, _ := rule["description"].(string)
	opts.Description = description

	remoteIPPrefix := rule["remote_ip_prefix"].(string)
	if remoteIPPrefix != "" {
		opts.RemoteIPPrefix = remoteIPPrefix
	}

	return opts, nil
}

// validatePortRange checks the validity of the port range specified in a security group rule for a given network protocol.
//
// Returns:
// - A pointer portRangeMin (which is an int alias) for the minimum port value, or nil if not applicable.
// - A pointer portRangeMax (which is an int alias) for the maximum port value, or nil if not applicable.
// - An error if any validation fails, or nil if all validations pass.
func validatePortRange(protocol edgecloudV2.SecurityGroupRuleProtocol, rule map[string]interface{}) (*portRangeMin, *portRangeMax, error) {
	portRangeMin := rule["port_range_min"].(portRangeMin)
	portRangeMax := rule["port_range_max"].(portRangeMax)

	if _, ok := networkProtocolWithPort[protocol]; ok {
		if portRangeMin == 0 || portRangeMax == 0 {
			return nil, nil, errors.New("port range min/max not specified")
		}

		if portRangeMin > portRangeMax {
			return nil, nil, errors.New("value of the port_range_min cannot be greater than port_range_max")
		}

		return &portRangeMin, &portRangeMax, nil
	}

	if portRangeMin != 0 || portRangeMax != 0 {
		return nil, nil, fmt.Errorf("%s network protocol does not support ports", protocol)
	}

	return nil, nil, nil
}
