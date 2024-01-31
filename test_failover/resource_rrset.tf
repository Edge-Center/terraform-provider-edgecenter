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
}

//
// example0: managing zone and records by TF using variables
//
variable "example_domain0" {
  type    = string
  default = "tftestzone4.com"
}

resource "edgecenter_dns_zone" "examplezone0" {
  name = var.example_domain0
}

resource "edgecenter_dns_zone_record" "example_rrset0" {
  zone   = edgecenter_dns_zone.examplezone0.name
  domain = edgecenter_dns_zone.examplezone0.name
  type   = "A"
  ttl    = 100

  filter {
    limit = 1
    type   = "is_healthy"
  }

  filter {
      type = "first_n"
      limit = 1
      strict = false
  }

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



