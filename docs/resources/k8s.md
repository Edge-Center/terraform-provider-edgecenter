---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_k8s Resource - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent k8s cluster with one default pool.
---

# edgecenter_k8s (Resource)

Represent k8s cluster with one default pool.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `fixed_network` (String)
- `fixed_subnet` (String) Subnet should has router
- `keypair` (String)
- `name` (String)
- `pool` (Block List, Min: 1, Max: 1) (see [below for nested schema](#nestedblock--pool))

### Optional

- `auto_healing_enabled` (Boolean)
- `external_dns_enabled` (Boolean)
- `last_updated` (String)
- `master_lb_floating_ip_enabled` (Boolean)
- `pods_ip_pool` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)
- `services_ip_pool` (String)
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `api_address` (String)
- `cluster_template_id` (String)
- `container_version` (String)
- `created_at` (String)
- `discovery_url` (String)
- `faults` (Map of String)
- `health_status` (String)
- `health_status_reason` (Map of String)
- `id` (String) The ID of this resource.
- `master_addresses` (List of String)
- `master_flavor_id` (String)
- `node_addresses` (List of String)
- `node_count` (Number)
- `status` (String)
- `status_reason` (String)
- `updated_at` (String)
- `user_id` (String)
- `version` (String)

<a id="nestedblock--pool"></a>
### Nested Schema for `pool`

Required:

- `flavor_id` (String)
- `max_node_count` (Number)
- `min_node_count` (Number)
- `name` (String)
- `node_count` (Number)

Optional:

- `docker_volume_size` (Number)
- `docker_volume_type` (String) Available value is 'standard', 'ssd_hiiops', 'cold', 'ultra'.

Read-Only:

- `created_at` (String)
- `stack_id` (String)
- `uuid` (String)


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `update` (String)

