---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_network Resource - edgecenter"
subcategory: ""
description: |-
  Represent network. A network is a software-defined network in a cloud computing infrastructure
---

# edgecenter_network (Resource)

Represent network. A network is a software-defined network in a cloud computing infrastructure

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_network" "network" {
  name       = "network_example"
  type       = "vxlan"
  region_id  = 1
  project_id = 1
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the network.

### Optional

- `create_router` (Boolean) Create external router to the network, default true
- `last_updated` (String) The timestamp of the last update (use with update context).
- `metadata_map` (Map of String) A map containing metadata, for example tags.
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.
- `type` (String) 'vlan' or 'vxlan' network type is allowed. Default value is 'vxlan'

### Read-Only

- `id` (String) The ID of this resource.
- `metadata_read_only` (List of Object) A list of read-only metadata items, e.g. tags. (see [below for nested schema](#nestedatt--metadata_read_only))
- `mtu` (Number) Maximum Transmission Unit (MTU) for the network. It determines the maximum packet size that can be transmitted without fragmentation.

<a id="nestedatt--metadata_read_only"></a>
### Nested Schema for `metadata_read_only`

Read-Only:

- `key` (String)
- `read_only` (Boolean)
- `value` (String)

## Import

Import is supported using the following syntax:

```shell
# import using <project_id>:<region_id>:<network_id> format
terraform import edgecenter_network.metwork1 1:6:447d2959-8ae0-4ca0-8d47-9f050a3637d7
```
