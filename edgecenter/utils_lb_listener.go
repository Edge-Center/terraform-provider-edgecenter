package edgecenter

import (
	"context"
	"errors"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func GetListenerL7PolicyUUIDS(ctx context.Context, client *edgecloudV2.Client, listenerID string) ([]string, error) {
	l7Policies, err := utilV2.L7PoliciesListByListenerID(ctx, client, listenerID)
	if err != nil && !errors.Is(err, utilV2.ErrL7PoliciesNotFound) {
		return nil, err
	}
	assignedToListenerL7Policies := make([]string, 0, len(l7Policies))
	for _, policy := range l7Policies {
		assignedToListenerL7Policies = append(assignedToListenerL7Policies, policy.ID)
	}
	return assignedToListenerL7Policies, nil
}
