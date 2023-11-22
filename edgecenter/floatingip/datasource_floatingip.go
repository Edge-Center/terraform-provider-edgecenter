package floatingip

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterFloatingIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterFloatingIPRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the project",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the region",
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "floating IP uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "floating_ip_address"},
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "floating IP address assigned to the resource, must be a valid IP address",
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
				ExactlyOneOf: []string{"id", "floating_ip_address"},
			},
			// computed attributes
		},
	}
}

func dataSourceEdgeCenterFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundFloatingIP *edgecloud.FloatingIP

	if id, ok := d.GetOk("id"); ok {
		floatingIP, _, err := client.Floatingips.Get(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundFloatingIP = floatingIP
	} else if floatingIPAddress, ok := d.GetOk("floating_ip_address"); ok {
		floatingIP, err := util.FloatingIPByIPAddress(ctx, client, floatingIPAddress.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundFloatingIP = floatingIP
	} else {
		return diag.Errorf("Error: specify either a floating_ip_address or id to lookup the floating ip")
	}

	d.SetId(foundFloatingIP.ID)
	d.Set("floating_ip_address", foundFloatingIP.FloatingIPAddress)

	return nil
}
