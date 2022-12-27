---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_subnet Data Source - terraform-provider-edgecenter"
subcategory: ""
description: |-
  
---

# edgecenter_subnet (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `metadata_k` (String)
- `metadata_kv` (Map of String)
- `network_id` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)

### Read-Only

- `cidr` (String)
- `connect_to_network_router` (Boolean)
- `dns_nameservers` (List of String)
- `enable_dhcp` (Boolean)
- `gateway_ip` (String)
- `host_routes` (List of Object) (see [below for nested schema](#nestedatt--host_routes))
- `id` (String) The ID of this resource.
- `metadata_read_only` (List of Object) (see [below for nested schema](#nestedatt--metadata_read_only))

<a id="nestedatt--host_routes"></a>
### Nested Schema for `host_routes`

Read-Only:

- `destination` (String)
- `nexthop` (String)


<a id="nestedatt--metadata_read_only"></a>
### Nested Schema for `metadata_read_only`

Read-Only:

- `key` (String)
- `read_only` (Boolean)
- `value` (String)

