package edgecenter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// getServerGroup retrieves a server group from the edge cloud service.
// It attempts to find the server group either by its ID or by its name.
func getServerGroup(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.ServerGroup, error) {
	var (
		serverGroup *edgecloudV2.ServerGroup
		err         error
	)

	name := d.Get(NameField).(string)
	srvGroupID := d.Get(IDField).(string)

	switch {
	case srvGroupID != "":
		serverGroup, _, err = clientV2.ServerGroups.Get(ctx, srvGroupID)
		if err != nil {
			return nil, err
		}
	default:
		serverGroups, _, err := clientV2.ServerGroups.List(ctx)
		if err != nil {
			return nil, err
		}

		foundServerGroups := make([]edgecloudV2.ServerGroup, 0, len(serverGroups))

		for _, sg := range serverGroups {
			if sg.Name == name {
				foundServerGroups = append(foundServerGroups, sg)
			}
		}

		switch {
		case len(foundServerGroups) == 0:
			return nil, errors.New("server group does not exist")

		case len(foundServerGroups) > 1:
			var message bytes.Buffer
			message.WriteString("Found server groups:\n")

			for _, fSG := range foundServerGroups {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", fSG.ID))
			}

			return nil, fmt.Errorf("multiple server groups found.\n %s.\n Use server group ID instead of name", message.String())
		}

		serverGroup = &foundServerGroups[0]
	}

	return serverGroup, err
}
