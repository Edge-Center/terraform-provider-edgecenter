terraform {
#   required_version = ">= 0.1.22"

  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
      version = "0.1.22"  # need to specify
    }
  }
}

provider "edgecenter" {
  permanent_api_token = "5623$84777d4986958985850cf76d81322219e6b051ca4825f72aae49d1aba1e2e8e56677bdd4b1c02f2a6bc3d77df9d2808bbc377070cf249d6c726683f8e8104ede"
}

//
// example0: managing zone and records by TF using variables
//
variable "example_domain0" {
  type    = string
  default = "examplezone.com"
}

resource "edgecenter_dns_zone" "examplezone0" {
  name = var.example_domain0
}

resource "edgecenter_dns_zone_record" "example_rrset0" {
  zone   = edgecenter_dns_zone.examplezone0.name
  domain = edgecenter_dns_zone.examplezone0.name
  type   = "A"
  ttl    = 100
  meta {
    failover {
        frequency = 10
        host = "test.ru"
        http_status_code = null
        method = "GET"
        port = 443
        protocol = "HTTP"
        regexp = ""
        timeout = 10
        tls = false
        url = "/"
        verify = false
    }
  }

  resource_record {
    content = "127.0.0.100"
  }
  resource_record {
    content = "127.0.0.200"
    // enabled = false
  }
}

//
// example1: managing zone outside of TF 
//
resource "edgecenter_dns_zone_record" "subdomain_examplezone" {
  zone   = "examplezone.com"
  domain = "subdomain.examplezone.com"
  type   = "TXT"
  ttl    = 10

  filter {
    type   = "geodistance"
    limit  = 1
    strict = true
  }

  resource_record {
    content = "1234"
    enabled = true

    meta {
      latlong    = [52.367, 4.9041]
      asn        = [12345]
      ip         = ["1.1.1.1"]
      notes      = ["notes"]
      continents = ["asia"]
      countries  = ["russia"]
      default    = true
    }
  }
}

resource "edgecenter_dns_zone_record" "subdomain_examplezone_mx" {
  zone   = "examplezone.com"
  domain = "subdomain.examplezone.com"
  type   = "MX"
  ttl    = 10

  resource_record {
    content = "10 mail.my.com."
    enabled = true
  }
}

resource "edgecenter_dns_zone_record" "subdomain_examplezone_caa" {
  zone   = "examplezone.com"
  domain = "subdomain.examplezone.com"
  type   = "CAA"
  ttl    = 10

  resource_record {
    content = "0 issue \"company.org; account=12345\""
    enabled = true
  }
}
