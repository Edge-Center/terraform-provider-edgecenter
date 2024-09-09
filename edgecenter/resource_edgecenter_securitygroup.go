package edgecenter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	SecurityGroupPoint = "securitygroups"
)

var ErrCannotDeleteSGRule = errors.New("error when deleting security group rule")

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		UpdateContext: resourceSecurityGroupUpdate,
		DeleteContext: resourceSecurityGroupDelete,
		Description:   "Represent SecurityGroups(Firewall)",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, sgID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(sgID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the security group.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A detailed description of the security group.",
			},
			"metadata_map": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "A map containing metadata, for example tags.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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
			"security_group_rules": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Firewall rules control what inbound(ingress) and outbound(egress) traffic is allowed to enter or leave a Instance. At least one 'egress' rule should be set",
				Set:         secGroupUniqueID,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"direction": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s'", edgecloudV2.SGRuleDirectionIngress, edgecloudV2.SGRuleDirectionEgress),
							ValidateDiagFunc: func(v interface{}, path cty.Path) diag.Diagnostics {
								val := v.(string)
								switch edgecloudV2.SecurityGroupRuleDirection(val) {
								case edgecloudV2.SGRuleDirectionIngress, edgecloudV2.SGRuleDirectionEgress:
									return nil
								}
								return diag.Errorf("wrong direction '%s', available value is '%s', '%s'", val, edgecloudV2.SGRuleDirectionIngress, edgecloudV2.SGRuleDirectionEgress)
							},
						},
						"ethertype": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available value is '%s', '%s'", edgecloudV2.EtherTypeIPv4, edgecloudV2.EtherTypeIPv6),
							ValidateDiagFunc: func(v interface{}, path cty.Path) diag.Diagnostics {
								val := v.(string)
								switch edgecloudV2.EtherType(val) {
								case edgecloudV2.EtherTypeIPv4, edgecloudV2.EtherTypeIPv6:
									return nil
								}
								return diag.Errorf("wrong ethertype '%s', available value is '%s', '%s'", val, edgecloudV2.EtherTypeIPv4, edgecloudV2.EtherTypeIPv6)
							},
						},
						"protocol": {
							Type:        schema.TypeString,
							Required:    true,
							Description: fmt.Sprintf("Available value is %s", strings.Join(edgecloudV2.SecurityGroupRuleProtocol("").StringList(), ",")),
						},
						"port_range_min": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"port_range_max": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      65535,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"remote_ip_prefix": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"updated_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_at": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"last_updated": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The timestamp of the last update (use with update context).",
			},
		},
	}
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start SecurityGroup creating")

	var valid bool
	vals := d.Get("security_group_rules").(*schema.Set).List()
	for _, val := range vals {
		rule := val.(map[string]interface{})
		if edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)) == edgecloudV2.SGRuleDirectionEgress {
			valid = true
			break
		}
	}
	if !valid {
		return diag.Errorf("at least one 'egress' rule should be set")
	}

	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("region_id", clientV2.Region)
	d.Set("project_id", clientV2.Project)

	rawRules := d.Get("security_group_rules").(*schema.Set).List()
	rules := make([]edgecloudV2.RuleCreateRequest, len(rawRules))
	for i, r := range rawRules {
		rule := r.(map[string]interface{})

		descr := rule["description"].(string)
		remoteIPPrefix := rule["remote_ip_prefix"].(string)

		sgrOpts := edgecloudV2.RuleCreateRequest{
			Direction:   edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)),
			EtherType:   edgecloudV2.EtherType(rule["ethertype"].(string)),
			Protocol:    edgecloudV2.SecurityGroupRuleProtocol(rule["protocol"].(string)),
			Description: &descr,
		}

		if remoteIPPrefix != "" {
			sgrOpts.RemoteIPPrefix = &remoteIPPrefix
		}

		portRangeMin := rule["port_range_min"].(int)
		portRangeMax := rule["port_range_max"].(int)

		if portRangeMin > portRangeMax {
			return diag.FromErr(fmt.Errorf("value of the port_range_min cannot be greater than port_range_max"))
		}

		sgrOpts.PortRangeMax = &portRangeMax
		sgrOpts.PortRangeMin = &portRangeMin

		rules[i] = sgrOpts
	}

	createSecurityGroupOpts := &edgecloudV2.SecurityGroupCreateRequestInner{}
	createSecurityGroupOpts.Name = d.Get("name").(string)
	createSecurityGroupOpts.SecurityGroupRules = rules

	if metadataRaw, ok := d.GetOk("metadata_map"); ok {
		metadataMap, err := MapInterfaceToMapString(metadataRaw)
		if err != nil {
			return diag.FromErr(err)
		}
		createSecurityGroupOpts.Metadata = *metadataMap
	}

	opts := edgecloudV2.SecurityGroupCreateRequest{
		SecurityGroup: *createSecurityGroupOpts,
	}
	descr := d.Get("description").(string)
	if descr != "" {
		opts.SecurityGroup.Description = &descr
	}

	sg, _, err := clientV2.SecurityGroups.Create(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(sg.ID)

	resourceSecurityGroupRead(ctx, d, m)
	log.Printf("[DEBUG] Finish SecurityGroup creating (%s)", sg.ID)

	return diags
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start SecurityGroup reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	sg, _, err := clientV2.SecurityGroups.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("region_id", sg.RegionID)
	d.Set("project_id", sg.ProjectID)
	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	metadataMap := make(map[string]string)
	metadataReadOnly := make([]map[string]interface{}, 0, len(sg.Metadata))

	if len(sg.Metadata) > 0 {
		for _, metadataItem := range sg.Metadata {
			metadataMap[metadataItem.Key] = metadataItem.Value
			metadataReadOnly = append(metadataReadOnly, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
	}

	if err := d.Set("metadata_map", metadataMap); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	newSgRules := make([]interface{}, len(sg.SecurityGroupRules))
	for i, sgr := range sg.SecurityGroupRules {
		log.Printf("rules: %+v", sgr)
		r := make(map[string]interface{})
		r["id"] = sgr.ID
		r["direction"] = sgr.Direction.String()

		if sgr.EtherType != nil {
			r["ethertype"] = sgr.EtherType.String()
		}

		r["protocol"] = edgecloudV2.SGRuleProtocolANY
		if sgr.Protocol != nil {
			r["protocol"] = sgr.Protocol.String()
		}

		r["port_range_max"] = 65535
		if sgr.PortRangeMax != nil {
			r["port_range_max"] = *sgr.PortRangeMax
		}
		r["port_range_min"] = 1
		if sgr.PortRangeMin != nil {
			r["port_range_min"] = *sgr.PortRangeMin
		}

		r["description"] = ""
		if sgr.Description != nil {
			r["description"] = *sgr.Description
		}

		r["remote_ip_prefix"] = ""
		if sgr.RemoteIPPrefix != nil {
			r["remote_ip_prefix"] = *sgr.RemoteIPPrefix
		}

		r["updated_at"] = sgr.UpdatedAt
		r["created_at"] = sgr.CreatedAt

		newSgRules[i] = r
	}

	if err := d.Set("security_group_rules", schema.NewSet(secGroupUniqueID, newSgRules)); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish SecurityGroup reading")

	return diags
}

func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start SecurityGroup updating")
	var valid bool
	vals := d.Get("security_group_rules").(*schema.Set).List()
	for _, val := range vals {
		rule := val.(map[string]interface{})
		if edgecloudV2.SecurityGroupRuleDirection(rule["direction"].(string)) == edgecloudV2.SGRuleDirectionEgress {
			valid = true
			break
		}
	}
	if !valid {
		return diag.Errorf("at least one 'egress' rule should be set")
	}

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	gid := d.Id()

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		req := &edgecloudV2.SecurityGroupUpdateRequest{
			Name:         newName,
			ChangedRules: []edgecloudV2.ChangedRules{},
		}
		_, _, err := clientV2.SecurityGroups.Update(ctx, gid, req)
		if err != nil {
			return diag.Errorf("Error updating security group name: %s", err)
		}
		log.Printf("[DEBUG] SecurityGroup name updated to: %s", newName)
	}

	if d.HasChange("security_group_rules") {
		oldRulesRaw, newRulesRaw := d.GetChange("security_group_rules")
		oldRules := oldRulesRaw.(*schema.Set)
		newRules := newRulesRaw.(*schema.Set)

		changedRule := make(map[string]bool)
		for _, r := range newRules.List() {
			rule := r.(map[string]interface{})
			rid := rule["id"].(string)
			if !oldRules.Contains(r) && rid == "" {
				opts := extractSecurityGroupRuleCreateRequestV2(r, gid)
				_, _, err = clientV2.SecurityGroups.RuleCreate(ctx, gid, &opts)
				if err != nil {
					return diag.FromErr(err)
				}

				continue
			}
			if rid != "" && !oldRules.Contains(r) {
				changedRule[rid] = true
				opts := extractSecurityGroupRuleUpdateRequestV2(r, gid)
				_, _, err = clientV2.SecurityGroups.RuleUpdate(ctx, gid, &opts)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}

		for _, r := range oldRules.List() {
			rule := r.(map[string]interface{})
			rid := rule["id"].(string)
			if !newRules.Contains(r) && !changedRule[rid] {
				_, resp, err := clientV2.SecurityGroups.RuleDelete(ctx, rid)
				if err != nil {
					return diag.FromErr(err)
				}
				if resp.StatusCode != http.StatusNoContent {
					return diag.FromErr(fmt.Errorf("sgRuleId: %s, error: %w", rid, ErrCannotDeleteSGRule))
				}
			}
		}
	}

	if d.HasChange("metadata_map") {
		_, nmd := d.GetChange("metadata_map")

		nmdMapString, err := MapInterfaceToMapString(nmd)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
		metaData := edgecloudV2.Metadata(*nmdMapString)

		_, err = clientV2.SecurityGroups.MetadataUpdate(ctx, gid, &metaData)
		if err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	d.Set("last_updated", time.Now().Format(time.RFC850))
	log.Println("[DEBUG] Finish SecurityGroup updating")

	return resourceSecurityGroupRead(ctx, d, m)
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start SecurityGroup deleting")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	sgID := d.Id()
	_, err = clientV2.SecurityGroups.Delete(ctx, sgID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of SecurityGroup deleting")

	return diags
}
