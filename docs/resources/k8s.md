---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_k8s Resource - edgecenter"
subcategory: ""
description: |-
  Represent k8s cluster with one default pool.
  WARNING: Resource "edgecenter_k8s" is deprecated and unavailable.
---

# edgecenter_k8s (Resource)

Represent k8s cluster with one default pool. 

 **WARNING:** Resource "edgecenter_k8s" is deprecated and unavailable.

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_k8s" "v" {
  project_id    = 1
  region_id     = 1
  version       = "1.25.11"
  name          = "tf-k8s"
  fixed_network = "6bf878c1-1ce4-47c3-a39b-6b5f1d79bf25"
  fixed_subnet  = "dc3a3ea9-86ae-47ad-a8e8-79df0ce04839"
  keypair       = "tf-keypair"
  pool {
    name               = "tf-pool"
    flavor_id          = "g1-standard-1-2"
    min_node_count     = 1
    max_node_count     = 2
    node_count         = 1
    docker_volume_size = 2
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `fixed_network` (String) Fixed network (uuid) associated with the Kubernetes cluster.
- `fixed_subnet` (String) Subnet (uuid) associated with the fixed network. Ensure there's a router on this subnet.
- `keypair` (String) The name of the keypair
- `name` (String) The name of the Kubernetes cluster.
- `pool` (Block List, Min: 1, Max: 1) Configuration details of the node pool in the Kubernetes cluster. (see [below for nested schema](#nestedblock--pool))
- `version` (String) The version of the Kubernetes cluster.

### Optional

- `auto_healing_enabled` (Boolean) Indicates whether auto-healing is enabled for the Kubernetes cluster. true by default.
- `last_updated` (String) The timestamp of the last update (use with update context).
- `master_lb_floating_ip_enabled` (Boolean) Flag indicating if the master LoadBalancer should have a floating IP.
- `pods_ip_pool` (String) IP pool to be used for pods within the Kubernetes cluster.
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.
- `services_ip_pool` (String) IP pool to be used for services within the Kubernetes cluster.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `api_address` (String) API endpoint address for the Kubernetes cluster.
- `cluster_template_id` (String) Template identifier from which the Kubernetes cluster was instantiated.
- `container_version` (String) The container runtime version used in the Kubernetes cluster.
- `created_at` (String) The timestamp when the Kubernetes cluster was created.
- `discovery_url` (String) URL used for node discovery within the Kubernetes cluster.
- `faults` (Map of String)
- `health_status` (String) Overall health status of the Kubernetes cluster.
- `health_status_reason` (Map of String)
- `id` (String) The ID of this resource.
- `master_addresses` (List of String) List of IP addresses for master nodes in the Kubernetes cluster.
- `master_flavor_id` (String) Identifier for the master node flavor in the Kubernetes cluster.
- `node_addresses` (List of String) List of IP addresses for worker nodes in the Kubernetes cluster.
- `node_count` (Number) Total number of nodes in the Kubernetes cluster.
- `status` (String) The current status of the Kubernetes cluster.
- `status_reason` (String) The reason for the current status of the Kubernetes cluster, if ERROR.
- `updated_at` (String) The timestamp when the Kubernetes cluster was updated.
- `user_id` (String) User identifier associated with the Kubernetes cluster.

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

## Import

Import is supported using the following syntax:

```shell
# import using <project_id>:<region_id>:<cluster_id> format
terraform import edgecenter_k8s.cluster1 1:6:447d2959-8ae0-4ca0-8d47-9f050a3637d7
```
