package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceInstance() *schema.Resource {
	return &schema.Resource{
		ReadContext:        dataSourceInstanceRead,
		Description:        "**WARNING:** Data source \"edgecenter_instance\" is deprecated and unavailable.\n Use \"edgecenter_instanceV2\" data-source instead.\n\n A cloud instance is a virtual machine in a cloud environment. Could be used with baremetal also.",
		DeprecationMessage: "!> **WARNING:** This data-source is deprecated and will be removed in the next major version. Use \"edgecenter_instanceV2\" data-source instead",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the instance.",
			},
			"flavor_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the flavor to be used for the instance, determining its compute and memory, for example 'g1-standard-2-4'.",
			},
			"volume": {
				Type:        schema.TypeSet,
				Computed:    true,
				Set:         volumeUniqueID,
				Description: "A set defining the volumes to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"volume_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"interface": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list defining the network interfaces to be attached to the instance.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"security_group": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A list of firewall configurations applied to the instance, defined by their id and name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"metadata": {
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
					},
				},
			},
			"flavor": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: `A map defining the flavor of the instance, for example, {"flavor_name": "g1-standard-2-4", "ram": 4096, ...}.`,
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the instance. This is computed automatically and can be used to track the instance's state.",
			},
			"vm_state": {
				Type:     schema.TypeString,
				Computed: true,
				Description: fmt.Sprintf(`The current virtual machine state of the instance, 
allowing you to start or stop the VM. Possible values are %s and %s.`, InstanceVMStateStopped, InstanceVMStateActive),
			},
			"addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of network addresses associated with the instance, for example "pub_net": [...].`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"net": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"addr": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceInstanceRead(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("data-source \"edgecenter_instance\" is deprecated and unavailable. Use edgecenter_instanceV2 data-source instead"))
}
