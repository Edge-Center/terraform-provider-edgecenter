package edgecenter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	LB7RuleCompareTypeField = "compare_type"
	LBL7RuleL7PolicyIDField = "l7policy_id"
	LBL7RuleValueField      = "value"
	LBL7RuleInvertField     = "invert"
	LBL7RuleCreateTimeout   = 10 * time.Minute
	LBL7RuleUpdateTimeout   = 10 * time.Minute
	LBL7RuleDeleteTimeout   = 10 * time.Minute
)

func resourceL7Rule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceL7RuleV2Create,
		ReadContext:   resourceL7RuleV2Read,
		UpdateContext: resourceL7RuleV2Update,
		DeleteContext: resourceL7RuleV2Delete,
		Description:   "An L7 Rule is a single, simple logical test which returns either true or false",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(LBL7RuleCreateTimeout),
			Update: schema.DefaultTimeout(LBL7RuleUpdateTimeout),
			Delete: schema.DefaultTimeout(LBL7RuleDeleteTimeout),
		},
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			TagsField: {
				Type:        schema.TypeList,
				Description: "A list of simple strings assigned to the resource.",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			TypeField: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf("The type of the L7 rule. Available types: \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\".",
					edgecloudV2.L7RuleTypeCookie, edgecloudV2.L7RuleTypeFyleType, edgecloudV2.L7RuleTypeHeader, edgecloudV2.L7RuleTypeHostName, edgecloudV2.L7RuleTypePath, edgecloudV2.L7RuleTypeSSLConnHasCert, edgecloudV2.L7RuleTypeSSLVerifyResult, edgecloudV2.L7RuleTypeSSLDNField),
				ValidateFunc: validation.StringInSlice([]string{
					string(edgecloudV2.L7RuleTypeCookie), string(edgecloudV2.L7RuleTypeFyleType), string(edgecloudV2.L7RuleTypeHeader), string(edgecloudV2.L7RuleTypeHostName), string(edgecloudV2.L7RuleTypePath), string(edgecloudV2.L7RuleTypeSSLConnHasCert), string(edgecloudV2.L7RuleTypeSSLVerifyResult), string(edgecloudV2.L7RuleTypeSSLDNField),
				}, true),
			},
			LB7RuleCompareTypeField: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf("The comparison type for the L7 rule. Available comparison types: \"%s\", \"%s\", \"%s\", \"%s\", \"%s\".",
					edgecloudV2.L7RuleCompareTypeContains, edgecloudV2.L7RuleCompareTypeStartsWith, edgecloudV2.L7RuleCompareTypeEndsWith, edgecloudV2.L7RuleCompareTypeEqualTo, edgecloudV2.L7RuleCompareTypeRegex),
				ValidateFunc: validation.StringInSlice([]string{
					string(edgecloudV2.L7RuleCompareTypeContains), string(edgecloudV2.L7RuleCompareTypeStartsWith), string(edgecloudV2.L7RuleCompareTypeEndsWith), string(edgecloudV2.L7RuleCompareTypeEqualTo), string(edgecloudV2.L7RuleCompareTypeRegex),
				}, true),
			},
			LBL7RuleL7PolicyIDField: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the L7 policy.",
			},
			LBL7PolicyListenerIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the listener.",
			},
			LBL7RuleValueField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The value to use for the comparison. For example, the file type to compare.",
				ValidateFunc: func(v interface{}, k string) (warnings []string, errors []error) { //nolint:nonamedreturns
					if len(v.(string)) == 0 {
						errors = append(errors, fmt.Errorf("'value' field should not be empty"))
					}
					return warnings, errors
				},
			},
			KeyField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The key to use for the comparison. For example, the name of the cookie to evaluate.",
			},
			LBL7RuleInvertField: {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "When `true` the logic of the rule is inverted.\n\nFor example, with `true`, equal to would become not equal to. Defaults to `false`.",
			},
			ProvisioningStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The provisioning status.",
			},
			OperatingStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating status.",
			},
		},
	}
}

func resourceL7RuleV2Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start L7 policy creating")

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	l7policyID := d.Get(LBL7RuleL7PolicyIDField).(string)
	ruleType := d.Get(TypeField).(string)
	key := d.Get(KeyField).(string)
	compareType := d.Get(LB7RuleCompareTypeField).(string)

	// Ensure the right combination of options have been specified.
	err = checkL7RuleType(ruleType, key)
	if err != nil {
		return diag.Errorf("Unable to create L7 Rule: %s", err)
	}

	createOpts := edgecloudV2.L7RuleCreateRequest{
		Type:        edgecloudV2.L7RuleType(ruleType),
		CompareType: edgecloudV2.L7RuleCompareType(compareType),
		Value:       d.Get("value").(string),
		Key:         key,
		Invert:      d.Get("invert").(bool),
	}

	if v, ok := d.GetOk(TagsField); ok {
		tags := v.([]interface{})
		for _, tag := range tags {
			createOpts.Tags = append(createOpts.Tags, tag.(string))
		}
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	log.Printf("[DEBUG] Attempting to create L7 Rule")
	result, _, err := clientV2.L7Rules.Create(ctx, l7policyID, &createOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := result.Tasks[0]

	taskInfo, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	taskResult, err := utilV2.ExtractTaskResultFromTask(taskInfo)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(taskResult.L7Rules[0])

	return resourceL7RuleV2Read(ctx, d, m)
}

func resourceL7RuleV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	l7policyID := d.Get(LBL7RuleL7PolicyIDField).(string)

	l7Rule, _, err := clientV2.L7Rules.Get(ctx, l7policyID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	l7Policy, _, err := clientV2.L7Policies.Get(ctx, l7policyID)
	if err != nil {
		return diag.FromErr(err)
	}

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

func resourceL7RuleV2Update(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	l7policyID := d.Get(LBL7RuleL7PolicyIDField).(string)
	ruleTypeStr := d.Get(TypeField).(string)
	key := d.Get(KeyField).(string)

	// Ensure the right combination of options have been specified.
	err = checkL7RuleType(ruleTypeStr, key)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := edgecloudV2.L7RuleUpdateRequest{}

	if d.HasChange("type") {
		ruleType := edgecloudV2.L7RuleType(ruleTypeStr)
		updateOpts.Type = &ruleType
	}
	if d.HasChange("compare_type") {
		ruleCompareType := edgecloudV2.L7RuleCompareType(d.Get("compare_type").(string))
		updateOpts.CompareType = &ruleCompareType
	}
	if d.HasChange("value") {
		value := d.Get("value").(string)
		updateOpts.Value = &value
	}
	if d.HasChange("key") {
		updateOpts.Key = &key
	}
	if d.HasChange("tags") {
		if v, ok := d.GetOk(TagsField); ok {
			tags := v.([]interface{})
			tagsToUpdate := make([]string, 0, len(tags))
			for _, tag := range tags {
				tagsToUpdate = append(tagsToUpdate, tag.(string))
			}
			updateOpts.Tags = &tagsToUpdate
		} else {
			updateOpts.Tags = &[]string{}
		}
	}
	if d.HasChange("invert") {
		invert := d.Get("invert").(bool)
		updateOpts.Invert = &invert
	}

	result, _, err := clientV2.L7Rules.Update(ctx, l7policyID, d.Id(), &updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := result.Tasks[0]

	_, err = utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceL7RuleV2Read(ctx, d, m)
}

func resourceL7RuleV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	l7policyID := d.Get("l7policy_id").(string)

	result, _, err := clientV2.L7Rules.Delete(ctx, l7policyID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	taskID := result.Tasks[0]

	_, err = utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
