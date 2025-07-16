provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_protection_resource_certificate" "protection_le_cert" {
  resource = edgecenter_protection_resource.protected_example_com.id
  ssl_type = "le"

  depends_on = [edgecenter_dns_zone_record.protected_resource_record]
}

resource "edgecenter_protection_resource" "protected_example_com" {
  name = "protected.example.com"
  tls  = ["1.2", "1.3"]
}

#
# Issuing LE certificate requires DNS record
#
resource "edgecenter_dns_zone" "examplezone" {
  name = "example.com"
}

resource "edgecenter_dns_zone_record" "protected_resource_record" {
  zone   = edgecenter_dns_zone.examplezone.name
  domain = edgecenter_protection_resource.protected_example_com.name
  type   = "A"
  ttl    = 100

  resource_record {
    content = edgecenter_protection_resource.protected_example_com.ip
  }
}
