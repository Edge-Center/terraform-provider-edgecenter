provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

#
# Set custom certificate
#

variable "cert" {
  type = string
}

variable "private_key" {
  type      = string
  sensitive = true
}

resource "edgecenter_protection_resource" "protected1_example_com" {
  name = "protected1.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_alias" "subdomain1" {
  resource = edgecenter_protection_resource.protected1_example_com.id
  name     = "subdomain.${edgecenter_protection_resource.protected1_example_com.name}"
}

resource "edgecenter_protection_resource_alias_certificate" "protection_cert_custom" {
  alias    = edgecenter_protection_resource_alias.subdomain1.id
  ssl_type = "custom"
  ssl_crt  = var.cert
  ssl_key  = var.private_key
}

#
# Set LE certificate
#

resource "edgecenter_protection_resource" "protected2_example_com" {
  name = "protected2.example.com"
  tls  = ["1.2", "1.3"]
}

resource "edgecenter_protection_resource_alias" "subdomain2" {
  resource = edgecenter_protection_resource.protected2_example_com.id
  name     = "subdomain.${edgecenter_protection_resource.protected2_example_com.name}"
}

resource "edgecenter_protection_resource_alias_certificate" "protection_cert_le" {
  alias    = edgecenter_protection_resource_alias.subdomain.id
  ssl_type = "le"

  depends_on = [edgecenter_dns_zone_record.protected_resource_record]
}

#
# Issuing LE certificate requires DNS record
#
resource "edgecenter_dns_zone" "examplezone" {
  name = "example.com"
}

resource "edgecenter_dns_zone_record" "protected_resource_record" {
  zone   = edgecenter_dns_zone.examplezone.name
  domain = edgecenter_protection_resource_alias.subdomain2.name
  type   = "A"
  ttl    = 100

  resource_record {
    content = edgecenter_protection_resource.protected2_example_com.ip
  }
}
