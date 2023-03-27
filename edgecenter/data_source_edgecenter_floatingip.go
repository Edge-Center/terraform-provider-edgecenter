package edgecenter

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/floatingip/v1/floatingips"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/utils"
)

func dataSourceFloatingIP() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceFloatingIPRead,
		Description: "A floating IP is a static IP address that points to one of your Instances. It allows you to redirect network traffic to any of your Instances in the same datacenter.",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"floating_ip_address": {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
					v := val.(string)
					ip := net.ParseIP(v)
					if ip != nil {
						return diag.Diagnostics{}
					}

					return diag.FromErr(fmt.Errorf("%q must be a valid ip, got: %s", key, v))
				},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"fixed_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"router_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"port_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"metadata_k": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"metadata_kv": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceFloatingIPRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start FloatingIP reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, FloatingIPsPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	ipAddr := d.Get("floating_ip_address").(string)
	metaOpts := &floatingips.ListOpts{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		metaOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		meta, err := utils.MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		metaOpts.MetadataKV = meta
	}

	ips, err := floatingips.ListAll(client, *metaOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var floatingIP floatingips.FloatingIPDetail
	for _, ip := range ips {
		if ip.FloatingIPAddress.String() == ipAddr {
			floatingIP = ip
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("floatingIP %s not found", ipAddr)
	}

	d.SetId(floatingIP.ID)
	if floatingIP.FixedIPAddress != nil {
		d.Set("fixed_ip_address", floatingIP.FixedIPAddress.String())
	} else {
		d.Set("fixed_ip_address", "")
	}

	d.Set("project_id", floatingIP.ProjectID)
	d.Set("region_id", floatingIP.RegionID)
	d.Set("status", floatingIP.Status)
	d.Set("port_id", floatingIP.PortID)
	d.Set("router_id", floatingIP.RouterID)
	d.Set("floating_ip_address", floatingIP.FloatingIPAddress.String())

	metadataReadOnly := PrepareMetadataReadonly(floatingIP.Metadata)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish FloatingIP reading")

	return diags
}
