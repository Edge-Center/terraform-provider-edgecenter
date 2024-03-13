---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_snapshot Data Source - edgecenter"
subcategory: ""
description: |-
  A snapshot is a feature that allows you to capture the current state of the instance or volume at a specific point in time
---

# edgecenter_snapshot (Data Source)

A snapshot is a feature that allows you to capture the current state of the instance or volume at a specific point in time

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

data "edgecenter_snapshot" "default" {
  name       = "default"
  region_id  = data.edgecenter_region.rg.id
  project_id = data.edgecenter_project.pr.id
}

output "view" {
  value = data.edgecenter_snapshot.default
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `name` (String) The name of the snapshot. Use only with uniq name.
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.
- `snapshot_id` (String) The ID of the snapshot.
- `volume_id` (String) The ID of the volume this snapshot was made from.

### Read-Only

- `created_at` (String) The datetime when the snapshot was created.
- `creator_task_id` (String) The task that created this entity.
- `description` (String) The description of the snapshot.
- `id` (String) The ID of this resource.
- `metadata` (Map of String) The metadata
- `size` (Number) The size of the snapshot, GiB.
- `status` (String) The status of the snapshot.
- `updated_at` (String) The datetime when the snapshot was last updated.