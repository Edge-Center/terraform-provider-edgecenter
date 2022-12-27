---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_lblistener Data Source - terraform-provider-edgecenter"
subcategory: ""
description: |-
  
---

# edgecenter_lblistener (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `loadbalancer_id` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `operating_status` (String)
- `pool_count` (Number)
- `protocol` (String) Available values is 'HTTP', 'HTTPS', 'TCP', 'UDP'
- `protocol_port` (Number)
- `provisioning_status` (String)

