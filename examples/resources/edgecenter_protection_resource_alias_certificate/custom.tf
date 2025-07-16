provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

variable "cert" {
  type = string
}

variable "private_key" {
  type      = string
  sensitive = true
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_alias" "subdomain" {
  resource = edgecenter_protection_resource.protected_example_com.id
  name     = "subdomain.${edgecenter_protection_resource.protected_example_com.name}"
}

resource "edgecenter_protection_resource_alias_certificate" "protection_custom_cert" {
  alias    = edgecenter_protection_resource_alias.subdomain.id
  ssl_type = "custom"
  ssl_crt  = var.cert
  ssl_key  = var.private_key
}
