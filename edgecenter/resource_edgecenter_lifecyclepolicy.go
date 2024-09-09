package edgecenter

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	LifecyclePolicyPoint = "lifecycle_policies"
	// Maybe move to utils and use for other resources.
	nameRegexString = `^[a-zA-Z0-9][a-zA-Z 0-9._\-]{1,61}[a-zA-Z0-9._]$`
)

// Maybe move to utils and use for other resources.
var nameRegex = regexp.MustCompile(nameRegexString)

func resourceLifecyclePolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLifecyclePolicyCreate,
		ReadContext:   resourceLifecyclePolicyRead,
		UpdateContext: resourceLifecyclePolicyUpdate,
		DeleteContext: resourceLifecyclePolicyDelete,
		Description:   "Represent lifecycle policy. Use to periodically take snapshots",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, lcpID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(lcpID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(nameRegex, ""),
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      edgecloudV2.LifeCyclePolicyStatusActive.String(),
				ValidateFunc: validation.StringInSlice(edgecloudV2.LifeCyclePolicyStatus("").StringList(), false),
			},
			"action": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      edgecloudV2.LifeCyclePolicyActionVolumeSnapshot.String(),
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(edgecloudV2.LifeCyclePolicyAction("").StringList(), false),
			},
			"volume": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of managed volumes",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.IsUUID,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"schedule": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_quantity": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 10000),
							Description:  "Maximum number of stored resources",
						},
						"interval": {
							Type:        schema.TypeList,
							MinItems:    1,
							MaxItems:    1,
							Description: "Use for taking actions with equal time intervals between them. Exactly one of interval and cron blocks should be provided",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"weeks": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: intervalScheduleParamDescription("week"),
									},
									"days": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: intervalScheduleParamDescription("day"),
									},
									"hours": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: intervalScheduleParamDescription("hour"),
									},
									"minutes": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: intervalScheduleParamDescription("minute"),
									},
								},
							},
							Optional: true,
						},
						"cron": {
							Type:        schema.TypeList,
							MinItems:    1,
							MaxItems:    1,
							Description: "Use for taking actions at specified moments of time. Exactly one of interval and cron blocks should be provided",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"timezone": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "UTC",
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"month": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "*",
										ValidateFunc:     validateCronField(1, 12),
										Description:      cronScheduleParamDescription(1, 12),
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"week": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "*",
										ValidateFunc:     validateCronField(1, 53),
										Description:      cronScheduleParamDescription(1, 53),
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"day": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "*",
										ValidateFunc:     validateCronField(1, 31),
										Description:      cronScheduleParamDescription(1, 31),
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"day_of_week": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "*",
										Description:      "Use lowercase three-letter abbreviations of weekdays comma-separated (e.g., 'mon,tue,wed') or '*' for any day.",
										ValidateFunc:     validateDaysOfWeek,
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"hour": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "*",
										ValidateFunc:     validateCronField(0, 23),
										Description:      cronScheduleParamDescription(0, 23),
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
									"minute": {
										Type:             schema.TypeString,
										Optional:         true,
										Default:          "0",
										ValidateFunc:     validateCronField(0, 59),
										Description:      cronScheduleParamDescription(0, 59),
										DiffSuppressFunc: suppressEquivalentCronDiffs,
									},
								},
							},
							Optional: true,
						},
						"resource_name_template": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "reserve snap of the volume {volume_id}",
							Description: "Used to name snapshots. {volume_id} is substituted with volume.id on creation",
						},
						"retention_time": {
							Type:        schema.TypeList,
							MinItems:    1,
							MaxItems:    1,
							Description: "If it is set, new resource will be deleted after time",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"weeks": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: retentionTimerParamDescription("week"),
									},
									"days": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: retentionTimerParamDescription("day"),
									},
									"hours": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: retentionTimerParamDescription("hour"),
									},
									"minutes": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: retentionTimerParamDescription("minute"),
									},
								},
							},
							Optional: true,
						},
						"id": {
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
			"user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceLifecyclePolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Start of LifecyclePolicy creating")
	opts, err := buildLifecyclePolicyCreateOptsV2(d)
	if err != nil {
		return diag.FromErr(err)
	}
	policy, _, err := clientV2.LifeCyclePolicies.Create(ctx, opts)
	if err != nil {
		return diag.Errorf("Error creating lifecycle policy: %s", err)
	}
	d.SetId(strconv.Itoa(policy.ID))
	log.Printf("[DEBUG] Finish of LifecyclePolicy %s creating", d.Id())

	return resourceLifecyclePolicyRead(ctx, d, m)
}

func resourceLifecyclePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	d.Set("region_id", clientV2.Region)
	d.Set("project_id", clientV2.Project)
	integerID, err := strconv.Atoi(id)
	if err != nil {
		return diag.Errorf("Error converting lifecycle policy ID to integer: %s", err)
	}

	log.Printf("[DEBUG] Start of LifecyclePolicy %s reading", id)
	policy, _, err := clientV2.LifeCyclePolicies.Get(ctx, integerID, &edgecloudV2.LifeCyclePolicyGetOptions{NeedVolumes: true})
	if err != nil {
		return diag.Errorf("Error getting lifecycle policy: %s", err)
	}

	_ = d.Set("name", policy.Name)
	_ = d.Set("status", policy.Status)
	_ = d.Set("action", policy.Action)
	_ = d.Set("user_id", policy.UserID)
	if err = d.Set("volume", flattenVolumesV2(policy.Volumes)); err != nil {
		return diag.Errorf("error setting lifecycle policy volumes: %s", err)
	}
	if err = d.Set("schedule", flattenSchedulesV2(policy.Schedules)); err != nil {
		return diag.Errorf("error setting lifecycle policy schedules: %s", err)
	}

	log.Printf("[DEBUG] Finish of LifecyclePolicy %s reading", id)

	return nil
}

func resourceLifecyclePolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	integerID, err := strconv.Atoi(id)
	if err != nil {
		return diag.Errorf("Error converting lifecycle policy ID to integer: %s", err)
	}

	log.Printf("[DEBUG] Start of LifecyclePolicy updating")
	if d.HasChanges("status", "name") {
		lifeCycleUpdateRequest := buildLifecyclePolicyUpdateOptsV2(d)
		_, _, err = clientV2.LifeCyclePolicies.Update(ctx, integerID, &lifeCycleUpdateRequest)
		if err != nil {
			return diag.Errorf("Error updating lifecycle policy: %s", err)
		}
	}

	if d.HasChange("volume") {
		oldVolumes, newVolumes := d.GetChange("volume")
		toRemove, toAdd := volumeSymmetricDifference(oldVolumes.(*schema.Set), newVolumes.(*schema.Set))
		_, _, err = clientV2.LifeCyclePolicies.RemoveVolumes(ctx, integerID, &edgecloudV2.LifeCyclePolicyRemoveVolumesRequest{VolumeIds: toRemove})
		if err != nil {
			return diag.Errorf("Error removing volumes from lifecycle policy: %s", err)
		}
		_, _, err = clientV2.LifeCyclePolicies.AddVolumes(ctx, integerID, &edgecloudV2.LifeCyclePolicyAddVolumesRequest{VolumeIds: toAdd})
		if err != nil {
			return diag.Errorf("Error adding volumes to lifecycle policy: %s", err)
		}
	}

	if d.HasChange("schedule") {
		oldSchedules, newSchedules := d.GetChange("schedule")

		oldScheduleIDs := make([]string, 0)
		for _, schedule := range oldSchedules.([]interface{}) {
			scheduleMap := schedule.(map[string]interface{})
			if id, ok := scheduleMap["id"].(string); ok && id != "" {
				oldScheduleIDs = append(oldScheduleIDs, id)
			}
		}

		if len(oldScheduleIDs) > 0 {
			req := &edgecloudV2.LifeCyclePolicyRemoveSchedulesRequest{ScheduleIDs: oldScheduleIDs}
			_, _, err = clientV2.LifeCyclePolicies.RemoveSchedules(ctx, integerID, req)
			if err != nil {
				return diag.Errorf("Error removing old schedules from lifecycle policy: %s", err)
			}
		}

		expanded, err := expandSchedulesV2(newSchedules.([]interface{}))
		if err != nil {
			return diag.Errorf("Error expanding new schedules: %s", err)
		}

		if len(expanded) > 0 {
			req := &edgecloudV2.LifeCyclePolicyAddSchedulesRequest{Schedules: expanded}
			_, _, err = clientV2.LifeCyclePolicies.AddSchedules(ctx, integerID, req)
			if err != nil {
				return diag.Errorf("Error adding new schedules to lifecycle policy: %s", err)
			}
		}
	}

	log.Printf("[DEBUG] Finish of LifecyclePolicy %v updating", integerID)

	return resourceLifecyclePolicyRead(ctx, d, m)
}

func resourceLifecyclePolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	integerID, err := strconv.Atoi(id)
	if err != nil {
		return diag.Errorf("Error converting lifecycle policy ID to integer: %s", err)
	}

	log.Printf("[DEBUG] Start of LifecyclePolicy %s deleting", id)
	_, err = clientV2.LifeCyclePolicies.Delete(ctx, integerID)
	if err != nil {
		return diag.Errorf("Error deleting lifecycle policy: %s", err)
	}
	d.SetId("")
	log.Printf("[DEBUG] Finish of LifecyclePolicy %s deleting", id)

	return nil
}

func expandIntervalScheduleV2(flat map[string]interface{}) *edgecloudV2.LifeCyclePolicyCreateIntervalScheduleRequest {
	return &edgecloudV2.LifeCyclePolicyCreateIntervalScheduleRequest{
		Weeks:   flat["weeks"].(int),
		Days:    flat["days"].(int),
		Hours:   flat["hours"].(int),
		Minutes: flat["minutes"].(int),
	}
}

func expandCronScheduleV2(flat map[string]interface{}) *edgecloudV2.LifeCyclePolicyCreateCronScheduleRequest {
	normalizeField := func(value interface{}) string {
		str, ok := value.(string)
		if !ok {
			return ""
		}

		if str == "*" {
			return str
		}

		return strings.Join(splitByCommaOrSpace(str), ",")
	}

	return &edgecloudV2.LifeCyclePolicyCreateCronScheduleRequest{
		Timezone:  normalizeField(flat["timezone"]),
		Week:      normalizeField(flat["week"]),
		DayOfWeek: normalizeField(flat["day_of_week"]),
		Month:     normalizeField(flat["month"]),
		Day:       normalizeField(flat["day"]),
		Hour:      normalizeField(flat["hour"]),
		Minute:    normalizeField(flat["minute"]),
	}
}

func expandRetentionTimerV2(flat []interface{}) *edgecloudV2.LifeCyclePolicyRetentionTimer {
	if len(flat) > 0 {
		rawRetention := flat[0].(map[string]interface{})
		return &edgecloudV2.LifeCyclePolicyRetentionTimer{
			Weeks:   rawRetention["weeks"].(int),
			Days:    rawRetention["days"].(int),
			Hours:   rawRetention["hours"].(int),
			Minutes: rawRetention["minutes"].(int),
		}
	}
	return nil
}

func expandScheduleV2(flat map[string]interface{}) (edgecloudV2.LifeCyclePolicyCreateScheduleRequest, error) {
	t := edgecloudV2.LifeCyclePolicyScheduleType("")
	intervalSlice := flat["interval"].([]interface{})
	cronSlice := flat["cron"].([]interface{})
	if len(intervalSlice)+len(cronSlice) != 1 {
		return nil, fmt.Errorf("exactly one of interval and cron blocks should be provided")
	}
	var expanded edgecloudV2.LifeCyclePolicyCreateScheduleRequest
	if len(intervalSlice) > 0 {
		t = edgecloudV2.LifeCyclePolicyScheduleTypeInterval
		expanded = expandIntervalScheduleV2(intervalSlice[0].(map[string]interface{}))
	} else {
		t = edgecloudV2.LifeCyclePolicyScheduleTypeCron
		expanded = expandCronScheduleV2(cronSlice[0].(map[string]interface{}))
	}
	expanded.SetCommonCreateScheduleOpts(edgecloudV2.LifeCyclePolicyCommonCreateScheduleRequest{
		Type:                 t,
		ResourceNameTemplate: flat["resource_name_template"].(string),
		MaxQuantity:          flat["max_quantity"].(int),
		RetentionTime:        expandRetentionTimerV2(flat["retention_time"].([]interface{})),
	})

	return expanded, nil
}

func expandSchedulesV2(flat []interface{}) ([]edgecloudV2.LifeCyclePolicyCreateScheduleRequest, error) {
	expanded := make([]edgecloudV2.LifeCyclePolicyCreateScheduleRequest, len(flat))
	for i, x := range flat {
		exp, err := expandScheduleV2(x.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		expanded[i] = exp
	}
	return expanded, nil
}

func expandVolumeIds(flat []interface{}) []string {
	expanded := make([]string, len(flat))
	for i, x := range flat {
		expanded[i] = x.(map[string]interface{})["id"].(string)
	}
	return expanded
}

func buildLifecyclePolicyCreateOptsV2(d *schema.ResourceData) (*edgecloudV2.LifeCyclePolicyCreateRequest, error) {
	schedules, err := expandSchedulesV2(d.Get("schedule").([]interface{}))
	if err != nil {
		return nil, err
	}
	opts := &edgecloudV2.LifeCyclePolicyCreateRequest{
		Name:      d.Get("name").(string),
		Status:    edgecloudV2.LifeCyclePolicyStatus(d.Get("status").(string)),
		Schedules: schedules,
		VolumeIds: expandVolumeIds(d.Get("volume").(*schema.Set).List()),
	}

	// Action is required field from API point of view, but optional for us
	if action, ok := d.GetOk("action"); ok {
		opts.Action = edgecloudV2.LifeCyclePolicyAction(action.(string))
	} else {
		opts.Action = edgecloudV2.LifeCyclePolicyActionVolumeSnapshot
	}

	return opts, nil
}

func volumeSymmetricDifference(oldVolumes, newVolumes *schema.Set) ([]string, []string) {
	toRemove := make([]string, 0)
	for _, v := range oldVolumes.List() {
		if !newVolumes.Contains(v) {
			toRemove = append(toRemove, v.(map[string]interface{})["id"].(string))
		}
	}
	toAdd := make([]string, 0)
	for _, v := range newVolumes.List() {
		if !oldVolumes.Contains(v) {
			toAdd = append(toAdd, v.(map[string]interface{})["id"].(string))
		}
	}

	return toRemove, toAdd
}

func buildLifecyclePolicyUpdateOptsV2(d *schema.ResourceData) edgecloudV2.LifeCyclePolicyUpdateRequest {
	opts := edgecloudV2.LifeCyclePolicyUpdateRequest{
		Name:   d.Get("name").(string),
		Status: edgecloudV2.LifeCyclePolicyStatus(d.Get("status").(string)),
	}
	return opts
}

func flattenIntervalScheduleV2(expanded edgecloudV2.LifeCyclePolicyIntervalSchedule) interface{} {
	return []map[string]int{{
		"weeks":   expanded.Weeks,
		"days":    expanded.Days,
		"hours":   expanded.Hours,
		"minutes": expanded.Minutes,
	}}
}

func flattenCronScheduleV2(expanded edgecloudV2.LifeCyclePolicyCronSchedule) interface{} {
	normalizeField := func(value string) string {
		if value == "*" {
			return value
		}
		return strings.Join(splitByCommaOrSpace(value), ",")
	}

	return []map[string]string{{
		"timezone":    expanded.Timezone,
		"week":        normalizeField(expanded.Week),
		"day_of_week": normalizeField(expanded.DayOfWeek),
		"month":       normalizeField(expanded.Month),
		"day":         normalizeField(expanded.Day),
		"hour":        normalizeField(expanded.Hour),
		"minute":      normalizeField(expanded.Minute),
	}}
}

func flattenRetentionTimerV2(expanded *edgecloudV2.LifeCyclePolicyRetentionTimer) interface{} {
	if expanded != nil {
		return []map[string]int{{
			"weeks":   expanded.Weeks,
			"days":    expanded.Days,
			"hours":   expanded.Hours,
			"minutes": expanded.Minutes,
		}}
	}
	return []interface{}{}
}

func flattenScheduleV2(expanded edgecloudV2.LifeCyclePolicySchedule) map[string]interface{} {
	common := expanded.GetCommonSchedule()
	flat := map[string]interface{}{
		"max_quantity":           common.MaxQuantity,
		"resource_name_template": common.ResourceNameTemplate,
		"retention_time":         flattenRetentionTimerV2(common.RetentionTime),
		"id":                     common.ID,
		"type":                   common.Type,
	}
	switch common.Type {
	case edgecloudV2.LifeCyclePolicyScheduleTypeInterval:
		flat["interval"] = flattenIntervalScheduleV2(expanded.(edgecloudV2.LifeCyclePolicyIntervalSchedule))
	case edgecloudV2.LifeCyclePolicyScheduleTypeCron:
		flat["cron"] = flattenCronScheduleV2(expanded.(edgecloudV2.LifeCyclePolicyCronSchedule))
	}

	return flat
}

func flattenSchedulesV2(expanded []edgecloudV2.LifeCyclePolicySchedule) []map[string]interface{} {
	flat := make([]map[string]interface{}, len(expanded))
	for i, x := range expanded {
		flat[i] = flattenScheduleV2(x)
	}
	return flat
}

func flattenVolumesV2(expanded []edgecloudV2.LifeCyclePolicyVolume) []map[string]string {
	flat := make([]map[string]string, len(expanded))
	for i, volume := range expanded {
		flat[i] = map[string]string{"id": volume.ID, "name": volume.Name}
	}
	return flat
}

func cronScheduleParamDescription(min, max int) string {
	return fmt.Sprintf("Either single asterisk or comma-separated list of integers (%v-%v)", min, max)
}

func intervalScheduleParamDescription(unit string) string {
	return fmt.Sprintf("Number of %ss to wait between actions", unit)
}

func retentionTimerParamDescription(unit string) string {
	return fmt.Sprintf("Number of %ss to wait before deleting snapshot", unit)
}

func validateDaysOfWeek(v interface{}, k string) ([]string, []error) {
	var errors []error

	value, ok := v.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
		return nil, errors
	}

	days := splitByCommaOrSpace(value)

	if len(days) == 1 && days[0] == "*" {
		return nil, nil
	}

	if len(days) > 7 {
		errors = append(errors, fmt.Errorf("too many days specified: %d. Maximum allowed is 7 days", len(days)))
		return nil, errors
	}

	validDaysMap := map[string]bool{
		"mon": true, "tue": true, "wed": true, "thu": true,
		"fri": true, "sat": true, "sun": true,
	}

	seenDays := make(map[string]bool)
	for _, day := range days {
		day = strings.ToLower(strings.TrimSpace(day))
		if day == "" {
			errors = append(errors, fmt.Errorf("empty day of week found. Please use lowercase three-letter abbreviations of weekdays (e.g., 'mon', 'tue')"))
			continue
		}
		if day == "*" {
			errors = append(errors, fmt.Errorf("'*' cannot be used with specific days. Use either '*' for all days or specify individual days"))
			continue
		}
		if !validDaysMap[day] {
			errors = append(errors, fmt.Errorf("invalid day of week: '%s'. Please use lowercase three-letter abbreviations of weekdays (e.g., 'mon', 'tue')", day))
			continue
		}
		if seenDays[day] {
			errors = append(errors, fmt.Errorf("duplicate day of week found: '%s'. Each day should be specified only once", day))
			continue
		}
		seenDays[day] = true
	}

	return nil, errors
}

func validateCronField(min, max int) func(v interface{}, k string) ([]string, []error) {
	return func(v interface{}, k string) ([]string, []error) {
		var errors []error

		value, ok := v.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return nil, errors
		}

		if value == "*" {
			return nil, nil
		}

		fields := splitByCommaOrSpace(value)

		for _, field := range fields {
			num, err := strconv.Atoi(field)
			if err != nil {
				errors = append(errors, fmt.Errorf("invalid value for %s: %s. Must be a number or '*'", k, field))
				continue
			}
			if num < min || num > max {
				errors = append(errors, fmt.Errorf("value %d for %s is out of range (%d-%d)", num, k, min, max))
			}
		}

		return nil, errors
	}
}

func splitByCommaOrSpace(s string) []string {
	commaFields := strings.Split(s, ",")

	result := make([]string, 0, len(commaFields))
	for _, field := range commaFields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func suppressEquivalentCronDiffs(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	normalizeField := func(value string) string {
		if value == "*" {
			return value
		}
		return strings.Join(splitByCommaOrSpace(value), ",")
	}
	return normalizeField(oldValue) == normalizeField(newValue)
}
