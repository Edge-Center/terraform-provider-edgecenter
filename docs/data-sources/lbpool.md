---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_lbpool Data Source - terraform-provider-edgecenter"
subcategory: ""
description: |-
  
---

# edgecenter_lbpool (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `listener_id` (String)
- `loadbalancer_id` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)

### Read-Only

- `health_monitor` (List of Object) (see [below for nested schema](#nestedatt--health_monitor))
- `id` (String) The ID of this resource.
- `lb_algorithm` (String) Available values is 'ROUND_ROBIN', 'LEAST_CONNECTIONS', 'SOURCE_IP', 'SOURCE_IP_PORT'
- `protocol` (String) Available values is 'HTTP' (currently work, other do not work on ed-8), 'HTTPS', 'TCP', 'UDP'
- `session_persistence` (List of Object) (see [below for nested schema](#nestedatt--session_persistence))

<a id="nestedatt--health_monitor"></a>
### Nested Schema for `health_monitor`

Read-Only:

- `delay` (Number)
- `expected_codes` (String)
- `http_method` (String)
- `id` (String)
- `max_retries` (Number)
- `max_retries_down` (Number)
- `timeout` (Number)
- `type` (String)
- `url_path` (String)


<a id="nestedatt--session_persistence"></a>
### Nested Schema for `session_persistence`

Read-Only:

- `cookie_name` (String)
- `persistence_granularity` (String)
- `persistence_timeout` (Number)
- `type` (String)

