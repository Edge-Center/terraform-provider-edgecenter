---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_useractions_subscription_amqp Data Source - edgecenter"
subcategory: ""
description: |-
  Data source provides access to user action logs and client subscription via AMQP.
---

# edgecenter_useractions_subscription_amqp (Data Source)

Data source provides access to user action logs and client subscription via AMQP.

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_useractions_subscription_amqp" "sub" {
}

output "view" {
  value = data.edgecenter_useractions_subscription_amqp.sub
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `connection_string` (String) A connection string of the following structure "scheme://username:password@host:port/virtual_host".
- `exchange` (String) Exchange name.
- `id` (String) The ID of this resource.
- `receive_child_client_events` (Boolean) Set to true if you would like to receive user action logs of all clients with reseller_id matching the current client_id. Defaults to false.
- `routing_key` (String) Routing key.
