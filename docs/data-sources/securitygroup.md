---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_securitygroup Data Source - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent SecurityGroups(Firewall)
---

# edgecenter_securitygroup (Data Source)

Represent SecurityGroups(Firewall)



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `metadata_k` (String)
- `metadata_kv` (Map of String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)

### Read-Only

- `description` (String)
- `id` (String) The ID of this resource.
- `metadata_read_only` (List of Object) (see [below for nested schema](#nestedatt--metadata_read_only))
- `security_group_rules` (Set of Object) Firewall rules control what inbound(ingress) and outbound(egress) traffic is allowed to enter or leave a Instance. At least one 'egress' rule should be set (see [below for nested schema](#nestedatt--security_group_rules))

<a id="nestedatt--metadata_read_only"></a>
### Nested Schema for `metadata_read_only`

Read-Only:

- `key` (String)
- `read_only` (Boolean)
- `value` (String)


<a id="nestedatt--security_group_rules"></a>
### Nested Schema for `security_group_rules`

Read-Only:

- `created_at` (String)
- `description` (String)
- `direction` (String)
- `ethertype` (String)
- `id` (String)
- `port_range_max` (Number)
- `port_range_min` (Number)
- `protocol` (String)
- `remote_ip_prefix` (String)
- `updated_at` (String)

