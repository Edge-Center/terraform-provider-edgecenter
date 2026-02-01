provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_rabbitmq" "example" {
  name     = "rabbitmq-example"
  enabled  = true
  place    = "region"
  entities = [10]
  ip       = "rabbitmq.example.com"
  port     = 5672
  username = "monitor"
  password = "example-password"
  vhost    = "/"
}
