package edgecenter

import (
	"fmt"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func checkL7RuleType(ruleType, key string) error {
	keyRequiredRuleTypesMap := map[edgecloudV2.L7RuleType]struct{}{edgecloudV2.L7RuleTypeCookie: {}, edgecloudV2.L7RuleTypeHeader: {}, edgecloudV2.L7RuleTypeSSLDNField: {}}
	if _, ok := keyRequiredRuleTypesMap[edgecloudV2.L7RuleType(ruleType)]; ok && key == "" {
		return fmt.Errorf("key attribute is required, when the L7 Rule type is %s", ruleType)
	} else if !ok && key != "" {
		return fmt.Errorf("key attribute must not be used, when the L7 Rule type is not %s", ruleType)
	}
	return nil
}

func checkL7RuleExistsInL7Policy(l7Policy edgecloudV2.L7Policy, l7RuleID string) bool {
	var l7RuleIsExists bool
	for _, l7PolicyRule := range l7Policy.Rules {
		if l7PolicyRule.ID == l7RuleID {
			l7RuleIsExists = true
		}
	}
	return l7RuleIsExists
}
