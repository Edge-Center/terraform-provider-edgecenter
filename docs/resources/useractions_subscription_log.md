---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_useractions_subscription_log Resource - edgecenter"
subcategory: ""
description: |-
  Resource provides access to user action logs and client subscription.
---

# edgecenter_useractions_subscription_log (Resource)

Resource provides access to user action logs and client subscription.

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_useractions_subscription_log" "subs" {
  auth_header_name  = "Authorization"
  auth_header_value = "Bearer eyJ0eXAi1.............Oi7Ix14"
  url               = "https://your-url.com/receive-user-action-messages"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `auth_header_name` (String) The name of the authorization header.
- `auth_header_value` (String) The value of the authorization header
- `url` (String) The URL to send user action logs for the specified client.

### Read-Only

- `id` (String) The ID of this resource.

## Import

Import is supported using the following syntax:

```shell
# import using <subscription_id> format
terraform import edgecenter_useractions_subscription_log.subs 123
```
