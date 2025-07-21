provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_origin" "primary" {
  resource     = edgecenter_protection_resource.protected_example_com.id
  ip           = "192.168.0.1"
  mode         = "primary"
  max_fails    = 2
  fail_timeout = 3
}

resource "edgecenter_protection_resource_origin" "backup" {
  resource = edgecenter_protection_resource.protected_example_com.id
  ip       = "192.168.0.2"
  mode     = "backup"
}

resource "edgecenter_protection_resource_origin" "down" {
  resource = edgecenter_protection_resource.protected_example_com.id
  ip       = "192.168.0.3"
  mode     = "down"
}
