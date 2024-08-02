package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func dataSourceInstancePortSecurity() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceInstancePortSecurityRead,
		Description: "Represent instance_port_security data_source.",

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},

			InstanceIDField: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "ID of the instance to which the port is connected.",
				ValidateFunc: validation.IsUUID,
			},

			PortSecurityDisabledField: {
				Type:        schema.TypeBool,
				Description: "Is the port_security feature disabled.",
				Computed:    true,
			},
			PortIDField: {
				Type:         schema.TypeString,
				Description:  "ID of the port.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			AllSecurityGroupIDsField: {
				Type:        schema.TypeSet,
				Description: "Set of all security groups IDs on this port.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceInstancePortSecurityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start instance_port_security reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIface, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}

	instancePort, err := utilV2.InstanceNetworkPortByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(portID)
	d.Set(PortSecurityDisabledField, !instanceIface.PortSecurityEnabled)

	if instanceIface.PortSecurityEnabled {
		sgIDs := make([]interface{}, len(instancePort.SecurityGroups))
		for idx, sg := range instancePort.SecurityGroups {
			sgIDs[idx] = sg.ID
		}
		d.Set(AllSecurityGroupIDsField, schema.NewSet(schema.HashString, sgIDs))
	}

	log.Println("[DEBUG] Finish instance_port_security reading")

	return diags
}
