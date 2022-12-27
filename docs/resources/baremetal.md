---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_baremetal Resource - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent baremetal instance
---

# edgecenter_baremetal (Resource)

Represent baremetal instance



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `flavor_id` (String)
- `interface` (Block List, Min: 1) (see [below for nested schema](#nestedblock--interface))

### Optional

- `app_config` (Map of String)
- `apptemplate_id` (String)
- `image_id` (String)
- `keypair_name` (String)
- `last_updated` (String)
- `metadata` (Block List, Deprecated) (see [below for nested schema](#nestedblock--metadata))
- `metadata_map` (Map of String)
- `name` (String)
- `name_template` (String)
- `name_templates` (List of String, Deprecated)
- `password` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `user_data` (String)
- `username` (String)

### Read-Only

- `addresses` (List of Object) (see [below for nested schema](#nestedatt--addresses))
- `flavor` (Map of String)
- `id` (String) The ID of this resource.
- `status` (String)
- `vm_state` (String)

<a id="nestedblock--interface"></a>
### Nested Schema for `interface`

Required:

- `type` (String) Available value is 'subnet', 'any_subnet', 'external', 'reserved_fixed_ip'

Optional:

- `existing_fip_id` (String)
- `fip_source` (String)
- `ip_address` (String)
- `is_parent` (Boolean) If not set will be calculated after creation. Trunk interface always attached first. Can't detach interface if is_parent true. Fields affect only on creation
- `network_id` (String) required if type is 'subnet' or 'any_subnet'
- `order` (Number) Order of attaching interface. Trunk interface always attached first, fields affect only on creation
- `port_id` (String) required if type is  'reserved_fixed_ip'
- `subnet_id` (String) required if type is 'subnet'


<a id="nestedblock--metadata"></a>
### Nested Schema for `metadata`

Required:

- `key` (String)
- `value` (String)


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)


<a id="nestedatt--addresses"></a>
### Nested Schema for `addresses`

Read-Only:

- `net` (List of Object) (see [below for nested schema](#nestedobjatt--addresses--net))

<a id="nestedobjatt--addresses--net"></a>
### Nested Schema for `addresses.net`

Read-Only:

- `addr` (String)
- `type` (String)

