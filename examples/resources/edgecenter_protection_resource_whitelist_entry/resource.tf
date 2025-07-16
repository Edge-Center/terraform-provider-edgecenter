provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_whitelist_entry" "whitelist1" {
  resource = edgecenter_protection_resource.protected_example_com.id
  ip       = "1.2.3.4"
}

resource "edgecenter_protection_resource_whitelist_entry" "whitelist2" {
  resource = edgecenter_protection_resource.protected_example_com.id
  ip       = "1.2.4.0/27"
}
