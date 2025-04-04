---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_reseller_networks Data Source - edgecenter"
subcategory: ""
description: |-
  !!! This data source has been created for resellers and only works with the reseller API key. !!!
  
  Returns the list of networks with subnet details that are available to the reseller and its clients in all regions.
  If the client_id and project_id parameters are not specified, the network or subnet is not owned by a reseller client or project.
---

# edgecenter_reseller_networks (Data Source)

!!! This data source has been created for resellers and only works with the reseller API key. !!!

	Returns the list of networks with subnet details that are available to the reseller and its clients in all regions.
	If the client_id and project_id parameters are not specified, the network or subnet is not owned by a reseller client or project.

## Example Usage

```terraform
provider "edgecenter" {
  # This data source has been created for resellers and only works with the reseller API key.
  permanent_api_token = "251$d3361.............1b35f26d8"
}


data "edgecenter_reseller_networks" "rnw" {
  shared   = false
  order_by = "name.desc"
  metadata_kv = {
    key_1 = "value_1"
  }

  metadata_k = ["key_1"]
}

output "view" {
  value = data.edgecenter_reseller_networks.rnw
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `metadata_k` (Set of String) Filter by metadata keys. Must be a valid JSON string. "metadata_k=["value", "sense"]"
- `metadata_kv` (Map of String) Filtration query opts, for example, {key = "value", key_1 = "value_1"}.
- `network_type` (String) Filter networks by the type of the network (vlan or vxlan).
- `order_by` (String) Order networks by transmitted fields and directions (name.asc).
- `shared` (Boolean) Can be used to only show networks with the shared state.

### Read-Only

- `id` (String) The ID of this resource.
- `networks` (List of Object) A list of read-only reseller networks. (see [below for nested schema](#nestedatt--networks))

<a id="nestedatt--networks"></a>
### Nested Schema for `networks`

Read-Only:

- `client_id` (Number)
- `created_at` (String)
- `creator_task_id` (String)
- `default` (Boolean)
- `external` (Boolean)
- `id` (String)
- `metadata` (List of Object) (see [below for nested schema](#nestedobjatt--networks--metadata))
- `mtu` (Number)
- `name` (String)
- `project_id` (Number)
- `region_id` (Number)
- `region_name` (String)
- `segmentation_id` (Number)
- `shared` (Boolean)
- `subnets` (List of Object) (see [below for nested schema](#nestedobjatt--networks--subnets))
- `task_id` (String)
- `type` (String)
- `updated_at` (String)

<a id="nestedobjatt--networks--metadata"></a>
### Nested Schema for `networks.metadata`

Read-Only:

- `key` (String)
- `read_only` (Boolean)
- `value` (String)


<a id="nestedobjatt--networks--subnets"></a>
### Nested Schema for `networks.subnets`

Read-Only:

- `available_ips` (Number)
- `cidr` (String)
- `dns_nameservers` (List of String)
- `enable_dhcp` (Boolean)
- `gateway_ip` (String)
- `has_router` (Boolean)
- `host_routes` (List of Object) (see [below for nested schema](#nestedobjatt--networks--subnets--host_routes))
- `id` (String)
- `name` (String)
- `total_ips` (Number)

<a id="nestedobjatt--networks--subnets--host_routes"></a>
### Nested Schema for `networks.subnets.host_routes`

Read-Only:

- `destination` (String)
- `nexthop` (String)
