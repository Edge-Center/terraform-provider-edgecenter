package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const LBL7RuleL7PolicyNameField = "l7policy_name"

func datasourceL7Rule() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceL7RuleRead,
		Description: "An L7 Rule is a single, simple logical test which returns either true or false",

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			IDField: {
				Type:         schema.TypeString,
				Description:  "The uuid of l7rule",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			LBL7RuleL7PolicyIDField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The ID of the L7 policy.",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{LBL7RuleL7PolicyIDField, LBL7RuleL7PolicyNameField},
			},
			LBL7RuleL7PolicyNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the L7 policy.",
				ExactlyOneOf: []string{LBL7RuleL7PolicyIDField, LBL7RuleL7PolicyNameField},
			},
			TagsField: {
				Type:        schema.TypeList,
				Description: "A list of simple strings assigned to the resource.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			TypeField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the L7 rule.",
			},
			LB7RuleCompareTypeField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The comparison type for the L7 rule.",
			},
			LBL7PolicyListenerIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the listener.",
			},
			LBL7RuleValueField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The value to use for the comparison. For example, the file type to compare.",
			},
			KeyField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The key to use for the comparison. For example, the name of the cookie to evaluate.",
			},
			LBL7RuleInvertField: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "When `true` the logic of the rule is inverted.\n\nFor example, with `true`, equal to would become not equal to. Defaults to `false`.",
			},
			ProvisioningStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The provisioning status",
			},
			OperatingStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating status",
			},
		},
	}
}

func datasourceL7RuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	l7ruleID := d.Get(IDField).(string)
	l7policyID := d.Get(LBL7RuleL7PolicyIDField).(string)
	l7policyName := d.Get(LBL7RuleL7PolicyNameField).(string)

	l7Policy, err := GetLBL7Policy(ctx, clientV2, l7policyID, l7policyName)
	if err != nil {
		return diag.FromErr(err)
	}

	l7Rule, _, err := clientV2.L7Rules.Get(ctx, l7policyID, l7ruleID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(l7Rule.ID)

	log.Printf("[DEBUG] Retrieved L7 Rule %s: %#v", d.Id(), l7Rule)

	d.Set(ProjectIDField, l7Rule.ProjectID)
	d.Set(RegionIDField, l7Rule.RegionID)
	d.Set(RegionNameField, l7Rule.Region)
	d.Set(LBL7RuleL7PolicyIDField, l7policyID)
	d.Set(LBL7PolicyListenerIDField, l7Policy.ListenerID)
	d.Set(TypeField, l7Rule.Type)
	d.Set(TagsField, l7Rule.Tags)
	d.Set(LB7RuleCompareTypeField, l7Rule.CompareType)
	d.Set(LBL7RuleValueField, l7Rule.Value)
	d.Set(KeyField, l7Rule.Key)
	d.Set(LBL7RuleInvertField, l7Rule.Invert)
	d.Set(OperatingStatusField, l7Rule.OperatingStatus)
	d.Set(ProvisioningStatusField, l7Rule.ProvisioningStatus)

	return nil
}
