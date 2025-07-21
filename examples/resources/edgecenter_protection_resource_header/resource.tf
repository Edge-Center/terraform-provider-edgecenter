provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_header" "header1" {
  resource = edgecenter_protection_resource.protected_example_com.id
  key      = "X-My-Header-1"
  value    = "Value 1"
}

resource "edgecenter_protection_resource_header" "header2" {
  resource = edgecenter_protection_resource.protected_example_com.id
  key      = "X-My-Header-2"
  value    = "Value 2"
}
