package edgecenter

import (
	"fmt"
	"strings"
)

func checkL7RuleType(ruleType, key string) error {
	keyRequired := []string{"COOKIE", "HEADER"}
	if (ruleType == "COOKIE" || ruleType == "HEADER") && key == "" {
		return fmt.Errorf("key attribute is required, when the L7 Rule type is %s", strings.Join(keyRequired, " or "))
	} else if (ruleType != "COOKIE" && ruleType != "HEADER") && key != "" {
		return fmt.Errorf("key attribute must not be used, when the L7 Rule type is not %s", strings.Join(keyRequired, " or "))
	}
	return nil
}
