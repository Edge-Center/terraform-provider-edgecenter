---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_loadbalancer Resource - terraform-provider-edgecenter"
subcategory: ""
description: |-
  Represent load balancer
---

# edgecenter_loadbalancer (Resource)

Represent load balancer



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `listener` (Block List, Min: 1, Max: 1) (see [below for nested schema](#nestedblock--listener))
- `name` (String)

### Optional

- `flavor` (String)
- `last_updated` (String)
- `project_id` (Number)
- `project_name` (String)
- `region_id` (Number)
- `region_name` (String)
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `vip_network_id` (String)
- `vip_subnet_id` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `vip_address` (String) Load balancer IP address

<a id="nestedblock--listener"></a>
### Nested Schema for `listener`

Required:

- `name` (String)
- `protocol` (String) Available values is 'HTTP' (currently work, other do not work on ed-8), 'HTTPS', 'TCP', 'UDP'
- `protocol_port` (Number)

Optional:

- `certificate` (String)
- `certificate_chain` (String)
- `insert_x_forwarded` (Boolean)
- `private_key` (String)
- `secret_id` (String)
- `sni_secret_id` (List of String)

Read-Only:

- `id` (String) The ID of this resource.


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)

