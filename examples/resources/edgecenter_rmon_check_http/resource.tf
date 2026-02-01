provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_check_http" "example" {
  name                  = "http-example"
  enabled               = true
  place                 = "country"
  entities              = [1]
  url                   = "https://example.com/health"
  method                = "get"
  accepted_status_codes = [200]
}
