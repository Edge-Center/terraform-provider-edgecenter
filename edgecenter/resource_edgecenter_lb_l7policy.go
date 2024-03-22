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
	LBL7PolicyCreateTimeout         = 2400 * time.Second
	LBL7PolicyUpdateTimeout         = 2400 * time.Second
	LBL7PolicyDeleteTimeout         = 2400 * time.Second
	LBL7PolicyRedirectHTTPCodeField = "redirect_http_code"
	LBL7PolicyRedirectPrefixField   = "redirect_prefix"
	LBL7PolicyRedirectURLField      = "redirect_url"
	LBL7PolicyRedirectPoolIDField   = "redirect_pool_id"
	LBL7PolicyTagsField             = "tags"
	LBL7PolicyPositionField         = "position"
	LBL7PolicyActionField           = "action"
	LBL7PolicyListenerIDField       = "listener_id"
	LBL7PolicyNameField             = "name"
	LBL7OperatingStatusField        = "operating_status"
	LBL7ProvisioningStatusField     = "provisioning_status"
)

func resourceL7Policy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceL7PolicyCreate,
		ReadContext:   resourceL7PolicyV2Read,
		UpdateContext: resourceL7PolicyV2Update,
		DeleteContext: resourceL7PolicyV2Delete,
		Description:   "An L7 Policy is a set of L7 rules, as well as a defined action applied to L7 network traffic. The action is taken if all the rules associated with the policy match",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, policyID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(policyID)

				return []*schema.ResourceData{d}, nil
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},

			RegionNameField: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},

			LBL7PolicyNameField: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The human-readable name of the policy",
			},

			LBL7PolicyActionField: {
				Description: "Enum: \"REDIRECT_PREFIX\" \"REDIRECT_TO_POOL\" \"REDIRECT_TO_URL\" \"REJECT\"\nThe action",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.StringInSlice([]string{
					"REDIRECT_TO_POOL", "REDIRECT_TO_URL", "REJECT", "REDIRECT_PREFIX",
				}, true),
			},

			LBL7PolicyListenerIDField: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The ID of the listener",
				ValidateFunc: validation.IsUUID,
			},

			LBL7PolicyPositionField: {
				Type:        schema.TypeInt,
				Description: "The position of this policy on the listener. Positions start at 1",
				Optional:    true,
				Computed:    true,
			},

			LBL7PolicyRedirectPoolIDField: {
				Type:          schema.TypeString,
				ConflictsWith: []string{LBL7PolicyRedirectURLField, LBL7PolicyRedirectPrefixField},
				Description:   "Requests matching this policy will be redirected to the pool with this ID. Only valid if the action is REDIRECT_TO_POOL",
				Optional:      true,
			},

			LBL7PolicyRedirectURLField: {
				Type:          schema.TypeString,
				ConflictsWith: []string{LBL7PolicyRedirectPoolIDField, LBL7PolicyRedirectPrefixField},
				Description:   "Requests matching this policy will be redirected to this URL. Only valid if the action is REDIRECT_TO_URL",
				Optional:      true,
				ValidateFunc:  validateURLFunc,
			},
			LBL7PolicyRedirectPrefixField: {
				Type:          schema.TypeString,
				ConflictsWith: []string{LBL7PolicyRedirectPoolIDField, LBL7PolicyRedirectURLField},
				Optional:      true,
				Description:   "Requests matching this policy will be redirected to this Prefix URL. Only valid if the action is REDIRECT_PREFIX",
				ValidateFunc:  validateURLFunc,
			},
			LBL7PolicyTagsField: {
				Type:        schema.TypeList,
				Description: "A list of simple strings assigned to the resource",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			LBL7PolicyRedirectHTTPCodeField: {
				Type:          schema.TypeInt,
				Description:   "Requests matching this policy will be redirected to the specified URL or Prefix URL with the HTTP response code. Valid if action is REDIRECT_TO_URL or REDIRECT_PREFIX. Valid options are 301, 302, 303, 307, or 308. Default is 302",
				ConflictsWith: []string{LBL7PolicyRedirectPoolIDField},
				ValidateFunc:  validation.IntInSlice([]int{301, 302, 303, 307, 308}),
				Optional:      true,
				Computed:      true,
			},
			LBL7ProvisioningStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The provisioning status",
			},
			LBL7OperatingStatusField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating status",
			},
			CreatedAtField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the L7 policy was created",
			},
			UpdatedAtField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the L7 policy was last updated",
			},
		},
	}
}

func resourceL7PolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start L7 policy creating")
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	listenerID := d.Get(LBL7PolicyListenerIDField).(string)
	actionStr := d.Get(LBL7PolicyActionField).(string)
	action := edgecloudV2.L7PolicyAction(actionStr)

	var redirectHTTPCodePtr *int
	if v, ok := d.GetOk(LBL7PolicyRedirectHTTPCodeField); ok {
		redirectHTTPCode := v.(int)
		redirectHTTPCodePtr = &redirectHTTPCode
	}

	// Ensure the right combination of options have been specified.
	err = checkL7PolicyAction(d, action, redirectHTTPCodePtr)
	if err != nil {
		return diag.Errorf("Unable to create L7 Policy: %s", err)
	}

	createOpts := edgecloudV2.L7PolicyCreateRequest{
		Action:     action,
		ListenerID: listenerID,
	}

	switch action {
	case edgecloudV2.L7PolicyActionRedirectToURL:
		createOpts.RedirectURL = d.Get(LBL7PolicyRedirectURLField).(string)
	case edgecloudV2.L7PolicyActionRedirectPrefix:
		createOpts.RedirectPrefix = d.Get(LBL7PolicyRedirectPrefixField).(string)
	case edgecloudV2.L7PolicyActionRedirectToPool:
		createOpts.RedirectPoolID = d.Get(LBL7PolicyRedirectPoolIDField).(string)
	case edgecloudV2.L7PolicyActionReject:
	}

	if v, ok := d.GetOk(LBL7PolicyNameField); ok {
		createOpts.Name = v.(string)
	}

	if v, ok := d.GetOk(LBL7PolicyTagsField); ok {
		tags := v.([]interface{})
		for _, tag := range tags {
			createOpts.Tags = append(createOpts.Tags, tag.(string))
		}
	}

	if v, ok := d.GetOk(LBL7PolicyPositionField); ok {
		createOpts.Position = v.(int)
	}

	if v, ok := d.GetOk(LBL7PolicyRedirectHTTPCodeField); ok {
		createOpts.RedirectHTTPCode = v.(int)
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	log.Printf("[DEBUG] Attempting to create L7 Policy")

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.L7Policies.Create, &createOpts, clientV2, LBL7PolicyCreateTimeout)
	if err != nil {
		return diag.Errorf("Error creating L7 Policy: %s", err)
	}

	l7PolicyID := taskResult.L7Polices[0]

	d.SetId(l7PolicyID)

	return resourceL7PolicyV2Read(ctx, d, m)
}

func resourceL7PolicyV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	l7Policy, _, err := clientV2.L7Policies.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Retrieved L7 Policy %s: %#v", d.Id(), l7Policy)

	d.Set(LBL7PolicyActionField, l7Policy.Action)
	d.Set(ProjectIDField, l7Policy.ProjectID)
	d.Set(RegionNameField, l7Policy.Region)
	d.Set(LBL7PolicyNameField, l7Policy.Name)
	d.Set(LBL7PolicyPositionField, l7Policy.Position)
	if l7Policy.RedirectHTTPCode != nil {
		d.Set(LBL7PolicyRedirectHTTPCodeField, l7Policy.RedirectHTTPCode)
	}
	d.Set(LBL7ProvisioningStatusField, l7Policy.ProvisioningStatus)
	d.Set(LBL7OperatingStatusField, l7Policy.OperatingStatus)
	d.Set(CreatedAtField, l7Policy.CreatedAt)
	d.Set(UpdatedAtField, l7Policy.UpdatedAt)

	return nil
}

func resourceL7PolicyV2Update(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	actionStr := d.Get(LBL7PolicyActionField).(string)
	action := edgecloudV2.L7PolicyAction(actionStr)

	isRedirectHTTPCodeNull := d.GetRawConfig().AsValueMap()[LBL7PolicyRedirectHTTPCodeField].IsNull()
	var redirectHTTPCodePtr *int
	if !isRedirectHTTPCodeNull {
		redirectHTTPCodeInt64, _ := d.GetRawConfig().GetAttr(LBL7PolicyRedirectHTTPCodeField).AsBigFloat().Int64()
		redirectHTTPCode := int(redirectHTTPCodeInt64)
		redirectHTTPCodePtr = &redirectHTTPCode
	}

	err = checkL7PolicyAction(d, action, redirectHTTPCodePtr)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOpts := edgecloudV2.L7PolicyUpdateRequest{Action: action}

	if v, ok := d.GetOk(LBL7PolicyNameField); ok {
		updateOpts.Name = v.(string)
	}
	if v, ok := d.GetOk(LBL7PolicyRedirectPoolIDField); ok {
		updateOpts.RedirectPoolID = v.(string)
	}
	if v, ok := d.GetOk(LBL7PolicyRedirectURLField); ok {
		updateOpts.RedirectURL = v.(string)
	}
	if v, ok := d.GetOk(LBL7PolicyRedirectPrefixField); ok {
		updateOpts.RedirectPrefix = v.(string)
	}
	if redirectHTTPCodePtr != nil {
		updateOpts.RedirectHTTPCode = *redirectHTTPCodePtr
	}
	if v, ok := d.GetOk(LBL7PolicyPositionField); ok {
		updateOpts.Position = v.(int)
	}
	if v, ok := d.GetOk(LBL7PolicyTagsField); ok {
		tags := v.([]interface{})
		for _, tag := range tags {
			updateOpts.Tags = append(updateOpts.Tags, tag.(string))
		}
	}

	log.Printf("[DEBUG] Updating L7 Policy %s with options: %#v", d.Id(), updateOpts)

	task, _, err := clientV2.L7Policies.Update(ctx, d.Id(), &updateOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := task.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID, LBL7PolicyUpdateTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceL7PolicyV2Read(ctx, d, m)
}

func resourceL7PolicyV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	id := d.Id()
	results, _, err := clientV2.L7Policies.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	task, err := utilV2.WaitAndGetTaskInfo(ctx, clientV2, taskID, LBL7PolicyDeleteTimeout)
	if err != nil {
		return diag.FromErr(err)
	}

	if task.State == edgecloudV2.TaskStateError {
		return diag.Errorf("cannot delete LBListener with ID: %s", id)
	}

	return nil
}

func checkL7PolicyAction(d *schema.ResourceData, action edgecloudV2.L7PolicyAction, redirectHTTPCode *int) error {
	redirectPoolID := d.Get(LBL7PolicyRedirectPoolIDField).(string)
	redirectURL := d.Get(LBL7PolicyRedirectURLField).(string)
	redirectPrefix := d.Get(LBL7PolicyRedirectPrefixField).(string)

	switch action {
	case edgecloudV2.L7PolicyActionRedirectPrefix:
		if redirectURL != "" || redirectPoolID != "" {
			return fmt.Errorf(
				"redirect_url and redirect_pool_id must be empty when action is set to %s", action)
		}
		if redirectPrefix == "" {
			return fmt.Errorf(
				"redirect_prefix must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionRedirectToPool:
		if redirectURL != "" || redirectPrefix != "" || redirectHTTPCode != nil {
			return fmt.Errorf(
				"redirect_url, redirect_prefix and redirect_http_code must be empty when action is set to %s", action)
		}
		if redirectPoolID == "" {
			return fmt.Errorf(
				"redirect_pool_id must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionRedirectToURL:
		if redirectPoolID != "" || redirectPrefix != "" {
			return fmt.Errorf(
				"redirect_prefix and redirect_pool_id must be empty when action is set to %s", action)
		}
		if redirectURL == "" {
			return fmt.Errorf(
				"redirect_url must be not empty when action is set to %s", action)
		}
	case edgecloudV2.L7PolicyActionReject:
		if redirectURL != "" || redirectPoolID != "" || redirectPrefix != "" || redirectHTTPCode != nil {
			return fmt.Errorf(
				"redirect_url, redirect_prefix, redirect_http_code and redirect_pool_id must be empty when action is set to %s", action)
		}
	}

	return nil
}
