provider "edgecenter" {
  permanent_api_token = "179...............45b54d"
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
  }

  resource_record {
    content = "127.0.0.100"
  }
  resource_record {
    content = "127.0.0.200"
    // enabled = false
  }
}

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

   meta {
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


  meta {
  }

  resource_record {
    content = "10 mail.my.com."
    enabled = true
  }
}

locals {
string = "0 issue \"company.org;account=12345\""
}

resource "edgecenter_dns_zone_record" "subdomain_examplezone_caa" {
  zone   = "examplezone.com"
  domain = "subdomain.examplezone.com"
  type   = "CAA"
  ttl    = 10

  meta {
  }

  resource_record {
    content = local.string
    enabled = true
  }
}
