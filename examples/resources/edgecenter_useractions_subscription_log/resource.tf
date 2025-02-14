provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_useractions_subscription_log" "subs" {
  auth_header_name  = "Authorization"
  auth_header_value = "Bearer eyJ0eXAi1.............Oi7Ix14"
  url               = "https://your-url.com/receive-user-action-messages"
}


