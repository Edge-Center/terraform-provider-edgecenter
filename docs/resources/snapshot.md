---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_snapshot Resource - edgecenter"
subcategory: ""
description: |-
  
---

# edgecenter_snapshot (Resource)



## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_snapshot" "snapshot" {
  project_id  = 1
  region_id   = 1
  name        = "snapshot example"
  volume_id   = "28e9edcb-1593-41fe-971b-da729c6ec301"
  description = "snapshot example description"
  metadata = {
    env = "test"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the snapshot.
- `volume_id` (String) The ID of the volume from which the snapshot was created.

### Optional

- `description` (String) A detailed description of the snapshot.
- `last_updated` (String) The timestamp of the last update (use with update context).
- `metadata` (Map of String)
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.

### Read-Only

- `id` (String) The ID of this resource.
- `metadata_read_only` (List of Object) A list of read-only metadata items, e.g. tags. (see [below for nested schema](#nestedatt--metadata_read_only))
- `size` (Number) The size of the snapshot in GB.
- `status` (String) The current status of the snapshot.

<a id="nestedatt--metadata_read_only"></a>
### Nested Schema for `metadata_read_only`

Read-Only:

- `key` (String)
- `read_only` (Boolean)
- `value` (String)

## Import

Import is supported using the following syntax:

```shell
# import using <project_id>:<region_id>:<snapshot_id> format
terraform import edgecenter_snapshot.snapshot1 1:6:447d2959-8ae0-4ca0-8d47-9f050a3637d7
```
