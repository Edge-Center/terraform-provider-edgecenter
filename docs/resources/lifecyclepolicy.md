---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_lifecyclepolicy Resource - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent lifecycle policy. Use to periodically take snapshots
---

# edgecenter_lifecyclepolicy (Resource)

Represent lifecycle policy. Use to periodically take snapshots



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `action` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)
- `schedule` (Block List) (see [below for nested schema](#nestedblock--schedule))
- `status` (String)
- `volume` (Block Set) List of managed volumes (see [below for nested schema](#nestedblock--volume))

### Read-Only

- `id` (String) The ID of this resource.
- `user_id` (Number)

<a id="nestedblock--schedule"></a>
### Nested Schema for `schedule`

Required:

- `max_quantity` (Number) Maximum number of stored resources

Optional:

- `cron` (Block List, Max: 1) Use for taking actions at specified moments of time. Exactly one of interval and cron blocks should be provided (see [below for nested schema](#nestedblock--schedule--cron))
- `interval` (Block List, Max: 1) Use for taking actions with equal time intervals between them. Exactly one of interval and cron blocks should be provided (see [below for nested schema](#nestedblock--schedule--interval))
- `resource_name_template` (String) Used to name snapshots. {volume_id} is substituted with volume.id on creation
- `retention_time` (Block List, Max: 1) If it is set, new resource will be deleted after time (see [below for nested schema](#nestedblock--schedule--retention_time))

Read-Only:

- `id` (String) The ID of this resource.
- `type` (String)

<a id="nestedblock--schedule--cron"></a>
### Nested Schema for `schedule.cron`

Optional:

- `day` (String) Either single asterisk or comma-separated list of integers (1-31)
- `day_of_week` (String) Either single asterisk or comma-separated list of integers (0-6)
- `hour` (String) Either single asterisk or comma-separated list of integers (0-23)
- `minute` (String) Either single asterisk or comma-separated list of integers (0-59)
- `month` (String) Either single asterisk or comma-separated list of integers (1-12)
- `timezone` (String)
- `week` (String) Either single asterisk or comma-separated list of integers (1-53)


<a id="nestedblock--schedule--interval"></a>
### Nested Schema for `schedule.interval`

Optional:

- `days` (Number) Number of days to wait between actions
- `hours` (Number) Number of hours to wait between actions
- `minutes` (Number) Number of minutes to wait between actions
- `weeks` (Number) Number of weeks to wait between actions


<a id="nestedblock--schedule--retention_time"></a>
### Nested Schema for `schedule.retention_time`

Optional:

- `days` (Number) Number of days to wait before deleting snapshot
- `hours` (Number) Number of hours to wait before deleting snapshot
- `minutes` (Number) Number of minutes to wait before deleting snapshot
- `weeks` (Number) Number of weeks to wait before deleting snapshot



<a id="nestedblock--volume"></a>
### Nested Schema for `volume`

Read-Only:

- `id` (String) The ID of this resource.
- `name` (String)

