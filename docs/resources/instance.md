---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_instance Resource - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent instance
---

# edgecenter_instance (Resource)

Represent instance



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `flavor_id` (String)
- `interface` (Block List, Min: 1) (see [below for nested schema](#nestedblock--interface))
- `volume` (Block Set, Min: 1) (see [below for nested schema](#nestedblock--volume))

### Optional

- `addresses` (Block List) (see [below for nested schema](#nestedblock--addresses))
- `allow_app_ports` (Boolean)
- `configuration` (Block List) (see [below for nested schema](#nestedblock--configuration))
- `flavor` (Map of String)
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
- `server_group` (String)
- `status` (String)
- `user_data` (String)
- `userdata` (String, Deprecated) **Deprecated**
- `username` (String)
- `vm_state` (String) Current vm state, use stopped to stop vm and active to start

### Read-Only

- `id` (String) The ID of this resource.
- `security_group` (List of Object) Firewalls list (see [below for nested schema](#nestedatt--security_group))

<a id="nestedblock--interface"></a>
### Nested Schema for `interface`

Optional:

- `existing_fip_id` (String)
- `fip_source` (String)
- `ip_address` (String)
- `network_id` (String) required if type is 'subnet' or 'any_subnet'
- `order` (Number) Order of attaching interface
- `port_id` (String) required if type is  'reserved_fixed_ip'
- `security_groups` (List of String) list of security group IDs
- `subnet_id` (String) required if type is 'subnet'
- `type` (String) Available value is 'subnet', 'any_subnet', 'external', 'reserved_fixed_ip'


<a id="nestedblock--volume"></a>
### Nested Schema for `volume`

Required:

- `source` (String) Currently available only 'existing-volume' value

Optional:

- `attachment_tag` (String)
- `boot_index` (Number) If boot_index==0 volumes can not detached
- `delete_on_termination` (Boolean)
- `image_id` (String)
- `name` (String)
- `size` (Number)
- `type_name` (String)
- `volume_id` (String)

Read-Only:

- `id` (String) The ID of this resource.


<a id="nestedblock--addresses"></a>
### Nested Schema for `addresses`

Required:

- `net` (Block List, Min: 1) (see [below for nested schema](#nestedblock--addresses--net))

<a id="nestedblock--addresses--net"></a>
### Nested Schema for `addresses.net`

Required:

- `addr` (String)
- `type` (String)



<a id="nestedblock--configuration"></a>
### Nested Schema for `configuration`

Required:

- `key` (String)
- `value` (String)


<a id="nestedblock--metadata"></a>
### Nested Schema for `metadata`

Required:

- `key` (String)
- `value` (String)


<a id="nestedatt--security_group"></a>
### Nested Schema for `security_group`

Read-Only:

- `id` (String)
- `name` (String)

