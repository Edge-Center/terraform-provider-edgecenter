provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_ping" "example" {
  name          = "ping-example"
  enabled       = true
  place         = "country"
  entities      = [1]
  ip            = "1.1.1.1"
  check_timeout = 2
}
