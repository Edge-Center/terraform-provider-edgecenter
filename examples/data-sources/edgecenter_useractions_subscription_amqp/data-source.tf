provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_useractions_subscription_amqp" "sub" {
}

output "view" {
  value = data.edgecenter_useractions_subscription_amqp.sub
}

