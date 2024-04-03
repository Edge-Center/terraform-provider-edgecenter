package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceL7Policy() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceL7PolicyV2Read,
		Description: "An L7 Policy is a set of L7 rules, as well as a defined action applied to L7 network traffic. The action is taken if all the rules associated with the policy match",
		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},

			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},

			IDField: {
				Type:          schema.TypeString,
				Description:   "The uuid of l7policy",
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},

			LBL7PolicyNameField: {
				ConflictsWith: []string{"id"},
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				Description:   "The human-readable name of the policy",
			},

			LBL7PolicyActionField: {
				Description: "Enum: \"REDIRECT_PREFIX\" \"REDIRECT_TO_POOL\" \"REDIRECT_TO_URL\" \"REJECT\"\nThe action",
				Type:        schema.TypeString,
				Computed:    true,
			},

			LBL7PolicyListenerIDField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the listener",
			},

			LBL7PolicyPositionField: {
				Type:        schema.TypeInt,
				Description: "The position of this policy on the listener. Positions start at 1",
				Computed:    true,
			},

			LBL7PolicyRedirectPoolIDField: {
				Type:        schema.TypeString,
				Description: "Requests matching this policy will be redirected to the pool with this ID. Only valid if the action is REDIRECT_TO_POOL",
				Computed:    true,
			},

			LBL7PolicyRedirectURLField: {
				Type:        schema.TypeString,
				Description: "Requests matching this policy will be redirected to this URL. Only valid if the action is REDIRECT_TO_URL",
				Computed:    true,
			},

			LBL7PolicyRedirectPrefixField: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Requests matching this policy will be redirected to this Prefix URL. Only valid if the action is REDIRECT_PREFIX",
			},

			LBL7PolicyTagsField: {
				Type:        schema.TypeSet,
				Description: "A list of simple strings assigned to the resource",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			LBL7PolicyRulesField: {
				Type:        schema.TypeSet,
				Description: "A set of l7rule uuids assigned to this l7policy",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			LBL7PolicyRedirectHTTPCodeField: {
				Type:        schema.TypeInt,
				Description: "Requests matching this policy will be redirected to the specified URL or Prefix URL with the HTTP response code. Valid if action is REDIRECT_TO_URL or REDIRECT_PREFIX. Valid options are 301, 302, 303, 307, or 308. Default is 302",
				Computed:    true,
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

func datasourceL7PolicyV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	l7Policy, err := GetLBL7Policy(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(l7Policy.ID)

	log.Printf("[DEBUG] Retrieved L7 Policy %s: %#v", d.Id(), l7Policy)

	d.Set(LBL7PolicyActionField, l7Policy.Action)
	d.Set(ProjectIDField, l7Policy.ProjectID)
	d.Set(RegionNameField, l7Policy.Region)
	d.Set(LBL7PolicyNameField, l7Policy.Name)
	d.Set(LBL7PolicyPositionField, l7Policy.Position)
	if l7Policy.RedirectURL != nil {
		d.Set(LBL7PolicyRedirectURLField, l7Policy.RedirectURL)
	}
	if l7Policy.RedirectPoolID != nil {
		d.Set(LBL7PolicyRedirectPoolIDField, l7Policy.RedirectPoolID)
	}
	if l7Policy.RedirectPrefix != nil {
		d.Set(LBL7PolicyRedirectPrefixField, l7Policy.RedirectPrefix)
	}
	if l7Policy.RedirectHTTPCode != nil {
		d.Set(LBL7PolicyRedirectHTTPCodeField, l7Policy.RedirectHTTPCode)
	}
	d.Set(LBL7PolicyListenerIDField, l7Policy.ListenerID)
	d.Set(TagsField, l7Policy.Tags)
	d.Set(ProvisioningStatusField, l7Policy.ProvisioningStatus)
	d.Set(OperatingStatusField, l7Policy.OperatingStatus)
	d.Set(CreatedAtField, l7Policy.CreatedAt)
	d.Set(UpdatedAtField, l7Policy.UpdatedAt)
	rules := make([]string, 0, len(l7Policy.Rules))
	for _, rule := range l7Policy.Rules {
		rules = append(rules, rule.ID)
	}
	d.Set(LBL7PolicyRulesField, rules)

	return nil
}
