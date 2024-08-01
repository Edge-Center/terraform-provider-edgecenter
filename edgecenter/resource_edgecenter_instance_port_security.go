package edgecenter

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	PortSecurityDisabledField         = "port_security_disabled"
	InstancePortSecurityCreateTimeout = 1200 * time.Second
	InstancePortSecurityReadTimeout   = 1200 * time.Second
	InstancePortSecurityDeleteTimeout = 1200 * time.Second
	InstancePortSecurityUpdateTimeout = 1200 * time.Second
)

func resourceInstancePortSecurity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstancePortSecurityCreate,
		ReadContext:   resourceInstancePortSecurityRead,
		UpdateContext: resourceInstancePortSecurityUpdate,
		DeleteContext: resourceInstancePortSecurityDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(InstancePortSecurityCreateTimeout),
			Read:   schema.DefaultTimeout(InstancePortSecurityReadTimeout),
			Update: schema.DefaultTimeout(InstancePortSecurityUpdateTimeout),
			Delete: schema.DefaultTimeout(InstancePortSecurityDeleteTimeout),
		},
		Description: "Represent instance_port_security resource",
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
				Type: schema.TypeBool,
				Description: "Is the port_security feature disabled. If this field has value \"true\", you can't use " +
					"\"security_groups\" field. You can't change port security of a public network port. When this field " +
					"has value \"true\" all security groups will be deleted. When this field switched back to value " +
					"\"false\" or deleted, default security group will be attached.",
				Computed: true,
				Optional: true,
			},
			PortIDField: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Description:  "ID of the instance network port.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			SecurityGroupsField: {
				Type:        schema.TypeSet,
				MaxItems:    1,
				Description: "Security groups.",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						SecurityGroupIDsField: {
							Type:        schema.TypeSet,
							Set:         schema.HashString,
							Description: "A set of security groups IDs that need to be attached.",
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						AllSecurityGroupIDsField: {
							Type: schema.TypeSet,
							Set:  schema.HashString,
							Description: "Set of all security groups IDs. This field has all security groups, " +
								"including those that were created outside of this resource (the default security group " +
								"and security groups created through the UI or API)",
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						OverwriteExistingField: {
							Type: schema.TypeBool,
							Description: "Whether to overwrite all security groups. If this field has value \"true\", " +
								"security groups that were created outside of this resource (the default security group " +
								"and security groups created through UI or API will be deleted and attached security groups specified in the attribute \"security_group_ids\" only)",
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func resourceInstancePortSecurityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start port_security creating")

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validatePortSecAttrs(d)
	if diags.HasError() {
		return diags
	}
	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)

	switch {
	case portSecurityDisabled && instanceIfacePort.PortSecurityEnabled:
		_, _, err = clientV2.Ports.DisablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
	case !portSecurityDisabled && !instanceIfacePort.PortSecurityEnabled:
		_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if portSecurityDisabled {
		d.SetId(portID)

		log.Println("[DEBUG] Finish instance_port_security creating")

		return resourceInstancePortSecurityRead(ctx, d, m)
	}

	sgsList := d.Get(SecurityGroupsField).(*schema.Set).List()
	switch len(sgsList) {
	case 0:
	default:
		sgsMap := sgsList[0].(map[string]interface{})
		sgsIDsRaw, sgsIDsOK := sgsMap[SecurityGroupIDsField]
		enforce := sgsMap[OverwriteExistingField].(bool)
		if enforce && sgsIDsOK {
			var sgsToRemove []interface{}

			instancePort, err := utilV2.InstanceNetworkPortByID(ctx, clientV2, instanceID, portID)
			if err != nil {
				return diag.FromErr(err)
			}
			if len(instancePort.SecurityGroups) != 0 {
				for _, sg := range instancePort.SecurityGroups {
					sgsToRemove = append(sgsToRemove, sg.ID)
				}
				err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgsToRemove)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
		sgsIDsSet := sgsIDsRaw.(*schema.Set)
		sgsIDsList := sgsIDsSet.List()
		err = AssignSecurityGroupsToInstancePort(ctx, clientV2, instanceID, portID, sgsIDsList)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(portID)

	log.Println("[DEBUG] Finish instance_port_security creating")

	return resourceInstancePortSecurityRead(ctx, d, m)
}

func resourceInstancePortSecurityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start instance_port_security reading")
	var diags diag.Diagnostics

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	clientV2, err := InitCloudClient(ctx, d, m)
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
	d.Set(PortSecurityDisabledField, !instanceIface.PortSecurityEnabled)

	sgsRaw, sgsRawOk := d.GetOk(SecurityGroupsField)
	if !sgsRawOk {
		log.Println("[DEBUG] Finish instance_port_security reading")
		return diags
	}

	sgsSetState := sgsRaw.(*schema.Set)
	sgsListState := sgsSetState.List()

	sgsMap := make(map[string]interface{}, 3)

	sgsMapState := sgsListState[0].(map[string]interface{})
	enforce := sgsMapState[OverwriteExistingField].(bool)
	sgsMap[OverwriteExistingField] = enforce

	sgIDsRaw, sgIDsRawOk := sgsMapState[SecurityGroupIDsField]
	allSgIDs := make([]interface{}, len(instancePort.SecurityGroups))
	for idx, sg := range instancePort.SecurityGroups {
		allSgIDs[idx] = sg.ID
	}
	allSgIDsSet := schema.NewSet(schema.HashString, allSgIDs)

	if sgIDsRawOk {
		sgIDsSet := sgIDsRaw.(*schema.Set)
		sgsMap[SecurityGroupIDsField] = allSgIDsSet.Intersection(sgIDsSet)
	}

	sgsMap[AllSecurityGroupIDsField] = allSgIDsSet

	sgsList := []interface{}{sgsMap}
	err = d.Set(SecurityGroupsField, schema.NewSet(sgsSetState.F, sgsList))
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish instance_port_security reading")

	return diags
}

func resourceInstancePortSecurityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start port_security updating")

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validatePortSecAttrs(d)
	if diags.HasError() {
		return diags
	}
	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)

	if d.HasChange(PortSecurityDisabledField) {
		instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
		if err != nil {
			return diag.FromErr(err)
		}

		switch {
		case portSecurityDisabled && instanceIfacePort.PortSecurityEnabled:
			_, _, err = clientV2.Ports.DisablePortSecurity(ctx, portID)
			if err != nil {
				return diag.FromErr(err)
			}
		case !portSecurityDisabled && !instanceIfacePort.PortSecurityEnabled:
			_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if portSecurityDisabled {
		log.Println("[DEBUG] Finish instance_port_security updating")

		return resourceInstancePortSecurityRead(ctx, d, m)
	}

	if d.HasChange(SecurityGroupsField) || d.HasChange(OverwriteExistingField) {
		var sgIDsToRemoveList []interface{}

		sgsOldRaw, sgsNewRaw := d.GetChange(SecurityGroupsField)
		sgsOldList, sgsNewList := sgsOldRaw.(*schema.Set).List(), sgsNewRaw.(*schema.Set).List()

		var sgsOldMap, sgsNewMap map[string]interface{}
		var enforce bool
		var sgIDsNewSet, sgIDsOldSet, allSgIDsOldSet *schema.Set

		switch len(sgsOldList) {
		case 0:
			instancePort, err := utilV2.InstanceNetworkPortByID(ctx, clientV2, instanceID, portID)
			if err != nil {
				return diag.FromErr(err)
			}
			allSgIDs := make([]interface{}, len(instancePort.SecurityGroups))
			for idx, sg := range instancePort.SecurityGroups {
				allSgIDs[idx] = sg.ID
			}
			allSgIDsOldSet = schema.NewSet(schema.HashString, allSgIDs)
			sgIDsOldSet = schema.NewSet(schema.HashString, []interface{}{})
		default:
			sgsOldMap = sgsOldList[0].(map[string]interface{})
			sgIDsOldSet = sgsOldMap[SecurityGroupIDsField].(*schema.Set)
			allSgIDsOldSet = sgsOldMap[AllSecurityGroupIDsField].(*schema.Set)
		}

		switch len(sgsNewList) {
		case 0:
			sgIDsNewSet = schema.NewSet(schema.HashString, []interface{}{})
		default:
			sgsNewMap = sgsNewList[0].(map[string]interface{})
			enforce = sgsNewMap[OverwriteExistingField].(bool)
			sgIDsNewSet = sgsNewMap[SecurityGroupIDsField].(*schema.Set)
		}

		switch enforce {
		case true:
			sgIDsToRemoveList = allSgIDsOldSet.Difference(sgIDsNewSet).List()
		default:
			sgIDsToRemoveList = sgIDsOldSet.Difference(sgIDsNewSet).List()
		}

		err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgIDsToRemoveList)
		if err != nil {
			return diag.FromErr(err)
		}

		sgsToAssignList := sgIDsNewSet.Difference(sgIDsOldSet).List()

		err = AssignSecurityGroupsToInstancePort(ctx, clientV2, instanceID, portID, sgsToAssignList)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	log.Println("[DEBUG] Finish instance_port_security updating")

	return resourceInstancePortSecurityRead(ctx, d, m)
}

func resourceInstancePortSecurityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start instance_port_security deleting")

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancel()

	var diags diag.Diagnostics
	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}

	if !instanceIfacePort.PortSecurityEnabled {
		_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	sgsRaw, ok := d.GetOk(SecurityGroupsField)
	if !ok {
		return diags
	}
	sgsList := sgsRaw.(*schema.Set).List()
	sgsMap := sgsList[0].(map[string]interface{})
	sgIDsSet := sgsMap[SecurityGroupIDsField].(*schema.Set)
	sgIDsList := sgIDsSet.List()
	err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgIDsList)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish instance_port_security deleting")

	return diags
}
