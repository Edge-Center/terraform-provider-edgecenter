---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_subnet Data Source - edgecenter"
subcategory: ""
description: |-
  
---

# edgecenter_subnet (Data Source)



## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_project" "pr" {
  name = "test"
}

data "edgecenter_region" "rg" {
  name = "ED-10 Preprod"
}

data "edgecenter_subnet" "tsn" {
  name       = "subtest"
  region_id  = data.edgecenter_region.rg.id
  project_id = data.edgecenter_project.pr.id
}

output "view" {
  value = data.edgecenter_subnet.tsn
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `id` (String) The ID of the subnet. Either 'id' or 'name' must be specified.
- `metadata_k` (String) Filtration query opts (only key).
- `metadata_kv` (Map of String) Filtration query opts, for example, {offset = "10", limit = "10"}
- `name` (String) The name of the subnet.
- `network_id` (String) The ID of the network to which this subnet belongs.
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.

### Read-Only

- `allocation_pools` (Set of Object) A list of allocation pools for DHCP. If omitted but DHCP or gateway settings are changed on update, pools are automatically reassigned. (see [below for nested schema](#nestedatt--allocation_pools))
- `cidr` (String) Represents the IP address range of the subnet.
- `connect_to_network_router` (Boolean) True if the network's router should get a gateway in this subnet. Must be explicitly 'false' when gateway_ip is null.
- `dns_nameservers` (List of String) List of DNS name servers for the subnet.
- `enable_dhcp` (Boolean) Enable DHCP for this subnet. If true, DHCP will be used to assign IP addresses to instances within this subnet.
- `gateway_ip` (String) The IP address of the gateway for this subnet.
- `host_routes` (List of Object) List of additional routes to be added to instances that are part of this subnet. (see [below for nested schema](#nestedatt--host_routes))
- `metadata_read_only` (List of Object) A list of read-only metadata items, e.g. tags. (see [below for nested schema](#nestedatt--metadata_read_only))

<a id="nestedatt--allocation_pools"></a>
### Nested Schema for `allocation_pools`

Read-Only:

- `end` (String)
- `start` (String)


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
