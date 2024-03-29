---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_servergroup Resource - edgecenter"
subcategory: ""
description: |-
  Represent server group resource
---

# edgecenter_servergroup (Resource)

Represent server group resource

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_servergroup" "default" {
  name       = "default"
  policy     = "affinity"
  region_id  = 1
  project_id = 1
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Displayed server group name
- `policy` (String) Server group policy. Available value is 'affinity', 'anti-affinity'

### Optional

- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.

### Read-Only

- `id` (String) The ID of this resource.
- `instances` (List of Object) Instances in this server group (see [below for nested schema](#nestedatt--instances))

<a id="nestedatt--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `instance_id` (String)
- `instance_name` (String)

## Import

Import is supported using the following syntax:

```shell
# import using <project_id>:<region_id>:<servergroup_id> format
terraform import edgecenter_servergroup.servergroup1 1:6:447d2959-8ae0-4ca0-8d47-9f050a3637d7
```
