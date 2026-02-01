provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_dns" "example" {
  name        = "dns-example"
  enabled     = true
  place       = "country"
  entities    = [1, 2]
  ip          = "example.com"
  resolver    = "8.8.8.8"
  record_type = "a"
}
