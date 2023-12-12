package loadbalancer

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func changeFloatingIP(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	oldFipRaw, newFipRaw := d.GetChange("floating_ip")
	oldFip, newFip := oldFipRaw.(string), newFipRaw.(string)

	if oldFip != "" {
		if _, _, err := client.Floatingips.UnAssign(ctx, oldFip); err != nil {
			return fmt.Errorf("error while unassign fip from loadbalancer: %w", err)
		}
	}

	if newFip != "" {
		assignFloatingIPRequest := &edgecloud.AssignFloatingIPRequest{
			PortID: d.Get("vip_port_id").(string),
		}

		if _, _, err := client.Floatingips.Assign(ctx, newFip, assignFloatingIPRequest); err != nil {
			return fmt.Errorf("error while assign fip to loadbalancer: %w", err)
		}
	}

	return nil
}
