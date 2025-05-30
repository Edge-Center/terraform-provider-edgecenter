package edgecenter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

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

// getLbListener retrieves a load balancer listener from the edge cloud service.
// It attempts to find the load balancer listener either by its ID or by its name.
func getLbListener(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Listener, error) {
	var (
		lbListener *edgecloudV2.Listener
		err        error
	)

	name := d.Get(NameField).(string)
	lbListenerID := d.Get(IDField).(string)

	switch {
	case lbListenerID != "":
		lbListener, _, err = clientV2.Loadbalancers.ListenerGet(ctx, lbListenerID)
		if err != nil {
			return nil, err
		}
	default:
		var opts edgecloudV2.ListenerListOptions

		lbID := d.Get(LoadbalancerIDField).(string)

		if lbID != "" {
			opts.LoadbalancerID = lbID
		}

		ls, _, err := clientV2.Loadbalancers.ListenerList(ctx, &opts)
		if err != nil {
			return nil, err
		}

		foundLbListeners := make([]edgecloudV2.Listener, 0, len(ls))

		for _, l := range ls {
			if l.Name == name {
				foundLbListeners = append(foundLbListeners, l)
			}
		}

		switch {
		case len(foundLbListeners) == 0:
			return nil, errors.New("load balancer listener does not exist")
		case len(foundLbListeners) > 1:
			var message bytes.Buffer
			message.WriteString("Found load balancer listeners:\n")

			for _, fLb := range foundLbListeners {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", fLb.ID))
			}

			return nil, fmt.Errorf("multiple load balancer listeners found.\n %s.\n Use load balancer ID instead of name", message.String())
		}

		lbListener = &foundLbListeners[0]
	}

	return lbListener, err
}
