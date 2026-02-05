provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_tcp" "example" {
  name     = "tcp-example"
  enabled  = true
  place    = "region"
  priority = "warning"
  entities = [10]
  ip       = "example.com"
  port     = 443
}
