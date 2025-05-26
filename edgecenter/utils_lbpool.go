package edgecenter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// getLBPool retrieves a load balancer pool from the edge cloud service.
// It attempts to find the load balancer pool either by its ID or by its name.
func getLBPool(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Pool, error) {
	var (
		pool *edgecloudV2.Pool
		err  error
	)

	name := d.Get(NameField).(string)
	poolId := d.Get(IDField).(string)

	switch {
	case poolId != "":
		pool, _, err = clientV2.Loadbalancers.PoolGet(ctx, poolId)
		if err != nil {
			return nil, err
		}
	default:
		var opts edgecloudV2.PoolListOptions

		lbID := d.Get(LoadbalancerIDField).(string)
		if lbID != "" {
			opts.LoadbalancerID = lbID
		}

		lID := d.Get(ListenerIDField).(string)
		if lbID != "" {
			opts.ListenerID = lID
		}

		pools, _, err := clientV2.Loadbalancers.PoolList(ctx, &opts)
		if err != nil {
			return nil, err
		}

		foundPools := make([]edgecloudV2.Pool, 0, len(pools))
		for _, p := range pools {
			if p.Name == name {
				foundPools = append(foundPools, p)
			}
		}

		switch {
		case len(foundPools) == 0:
			return nil, errors.New("load balancer pool does not exist")

		case len(foundPools) > 1:
			var message bytes.Buffer
			message.WriteString("Found load balancer pools:\n")

			for _, fpl := range foundPools {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", fpl.ID))
			}

			return nil, fmt.Errorf("multiple load balancer pools found.\n %s.\n Use load balancer pool ID instead of name", message.String())
		}

		pool = &foundPools[0]
	}

	return pool, nil
}
