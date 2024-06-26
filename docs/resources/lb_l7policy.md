---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_lb_l7policy Resource - edgecenter"
subcategory: ""
description: |-
  An L7 Policy is a set of L7 rules, as well as a defined action applied to L7 network traffic. The action is taken if all the rules associated with the policy match
---

# edgecenter_lb_l7policy (Resource)

An L7 Policy is a set of L7 rules, as well as a defined action applied to L7 network traffic. The action is taken if all the rules associated with the policy match

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_loadbalancerv2" "lb" {
  flavor       = "lb1-1-2"
  metadata_map = {}
  name         = "test-l7policy"
  project_id   = 1
  region_id    = 1
}

resource "edgecenter_lblistener" "listener" {
  project_id      = 1
  region_id       = 1
  name            = "test-l7policy"
  protocol        = "HTTP"
  protocol_port   = 80
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
}


resource "edgecenter_lb_l7policy" "l7policy" {
  name               = "test-policy"
  project_id         = 1
  region_id          = 1
  action             = "REDIRECT_PREFIX"
  listener_id        = edgecenter_lblistener.listener.id
  redirect_http_code = 303
  redirect_prefix    = "https://your-prefix.com/"
  tags               = ["aaa", "bbb", "ccc"]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `action` (String) Enum: "REDIRECT_PREFIX" "REDIRECT_TO_POOL" "REDIRECT_TO_URL" "REJECT"
The action.
- `listener_id` (String) The ID of the listener

### Optional

- `name` (String) The human-readable name of the policy
- `position` (Number) The position of this policy on the listener. Positions start at 1
- `project_id` (Number) The uuid of the project. Either 'project_id' or 'project_name' must be specified.
- `project_name` (String) The name of the project. Either 'project_id' or 'project_name' must be specified.
- `redirect_http_code` (Number) Requests matching this policy will be redirected to the specified URL or Prefix URL with the HTTP response code. Valid if action is REDIRECT_TO_URL or REDIRECT_PREFIX. Valid options are 301, 302, 303, 307, or 308. Default is 302
- `redirect_pool_id` (String) Requests matching this policy will be redirected to the pool with this ID. Only valid if the action is REDIRECT_TO_POOL
- `redirect_prefix` (String) Requests matching this policy will be redirected to this Prefix URL. Only valid if the action is REDIRECT_PREFIX
- `redirect_url` (String) Requests matching this policy will be redirected to this URL. Only valid if the action is REDIRECT_TO_URL
- `region_id` (Number) The uuid of the region. Either 'region_id' or 'region_name' must be specified.
- `region_name` (String) The name of the region. Either 'region_id' or 'region_name' must be specified.
- `tags` (Set of String) A list of simple strings assigned to the resource
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `created_at` (String) The datetime when the L7 policy was created
- `id` (String) The ID of this resource.
- `operating_status` (String) The operating status
- `provisioning_status` (String) The provisioning status
- `rules` (Set of String) A set of l7rule uuids assigned to this l7policy
- `updated_at` (String) The datetime when the L7 policy was last updated

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `update` (String)

## Import

Import is supported using the following syntax:

```shell
# import using <project_id>:<region_id>:<policy_id> format
terraform import edgecenter_lb_l7policy.lbpolicy1 1:6:447d2959-8ae0-4ca0-8d47-9f050a3637d7
```
