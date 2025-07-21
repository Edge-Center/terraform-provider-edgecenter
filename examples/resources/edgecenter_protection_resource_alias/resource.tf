provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_alias" "subdomain1" {
  resource = edgecenter_protection_resource.protected_example_com.id
  name     = "subdomain1.${edgecenter_protection_resource.protected_example_com.name}"
}

resource "edgecenter_protection_resource_alias" "subdomain2" {
  resource = edgecenter_protection_resource.protected_example_com.id
  name     = "subdomain2.${edgecenter_protection_resource.protected_example_com.name}"
}
