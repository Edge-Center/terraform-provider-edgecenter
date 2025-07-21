provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name              = "protected.example.com"
  tls               = ["1.2", "1.3"]
  www_redirect      = true
  waf               = true
  redirect_to_https = true
  active            = true
  geoip_mode        = "allow"
  geoip_list        = ["RU"]
}
