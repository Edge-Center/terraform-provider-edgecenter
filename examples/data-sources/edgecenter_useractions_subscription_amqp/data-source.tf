provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_useractions_subscription_amqp" "sub" {
}

output "view" {
  value = data.edgecenter_useractions_subscription_amqp.sub
}


data "edgecenter_useractions_subscription_amqp" "sub_for_client" {
  client_id = 123
}

output "sub_client_view" {
  value = data.edgecenter_useractions_subscription_amqp.sub_for_client
}

