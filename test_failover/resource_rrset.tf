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
  permanent_api_token = "12687$f95c4e7d9547e381deb7b6d499c63d045b3aeb94292c3a9e07685f7c78907d417d07f40bac848bb8b219c220e449d81fa43aee452e8254a99c2fd6b6f1808186"
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

    filter {
      limit = 1
      type   = "is_healthy"
    }

  meta {
    failover {
        frequency = 10
        host = "test.ru"
        method = "GET"
        port = 443
        protocol = "HTTP"
        regexp = ""
        timeout = 10
        tls = false
        url = "/"
        verify = false
        http_status_code = 200
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



