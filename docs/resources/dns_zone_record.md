---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "edgecenter_dns_zone_record Resource - edgecenter"
subcategory: ""
description: |-
  Represent DNS Zone Record resource. https://dns.edgecenter.ru/zones
---

# edgecenter_dns_zone_record (Resource)

Represent DNS Zone Record resource. https://dns.edgecenter.ru/zones

## Example Usage

```terraform
provider "edgecenter" {
  permanent_api_token = "179$...............45b54d"
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (String) A domain of DNS Zone Record resource.
- `meta` (Block List, Min: 1, Max: 1) A meta of DNS Zone Record resource. (see [below for nested schema](#nestedblock--meta))
- `resource_record` (Block Set, Min: 1) An array of contents with meta of DNS Zone Record resource. (see [below for nested schema](#nestedblock--resource_record))
- `type` (String) A type of DNS Zone Record resource.
- `zone` (String) A zone of DNS Zone Record resource.

### Optional

- `filter` (Block List) (see [below for nested schema](#nestedblock--filter))
- `ttl` (Number) A ttl of DNS Zone Record resource.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--meta"></a>
### Nested Schema for `meta`

Optional:

- `failover` (Block List, Max: 1) A failover meta of DNS Zone Record resource. (see [below for nested schema](#nestedblock--meta--failover))

<a id="nestedblock--meta--failover"></a>
### Nested Schema for `meta.failover`

Required:

- `frequency` (Number) A failover frequency of DNS Zone Record resource.
- `protocol` (String) A failover protocol of DNS Zone Record resource.
- `timeout` (Number) A failover timeout of DNS Zone Record resource.

Optional:

- `host` (String) A failover host of DNS Zone Record resource.
- `http_status_code` (Number) A failover http status code of DNS Zone Record resource.
- `method` (String) A failover method of DNS Zone Record resource.
- `port` (Number) A failover port of DNS Zone Record resource.
- `regexp` (String) A failover regexp of DNS Zone Record resource.
- `tls` (Boolean) A failover tls of DNS Zone Record resource.
- `url` (String) A failover url of DNS Zone Record resource.
- `verify` (Boolean) A failover verify of DNS Zone Record resource.



<a id="nestedblock--resource_record"></a>
### Nested Schema for `resource_record`

Required:

- `content` (String) A content of DNS Zone Record resource. (TXT: 'anyString', MX: '50 mail.company.io.', CAA: '0 issue "company.org; account=12345"')

Optional:

- `enabled` (Boolean) Manage of public appearing of DNS Zone Record resource.
- `meta` (Block Set, Max: 1) (see [below for nested schema](#nestedblock--resource_record--meta))

<a id="nestedblock--resource_record--meta"></a>
### Nested Schema for `resource_record.meta`

Optional:

- `asn` (List of Number) An asn meta (e.g. 12345) of DNS Zone Record resource.
- `continents` (List of String) Continents meta (e.g. Asia) of DNS Zone Record resource.
- `countries` (List of String) Countries meta (e.g. USA) of DNS Zone Record resource.
- `default` (Boolean) Fallback meta equals true marks records which are used as a default answer (when nothing was selected by specified meta fields).
- `ip` (List of String) An ip meta (e.g. 127.0.0.0) of DNS Zone Record resource.
- `latlong` (List of Number) A latlong meta (e.g. 27.988056, 86.925278) of DNS Zone Record resource.
- `notes` (List of String) A notes meta (e.g. Miami DC) of DNS Zone Record resource.



<a id="nestedblock--filter"></a>
### Nested Schema for `filter`

Required:

- `type` (String) A DNS Zone Record filter option that describe a name of filter.

Optional:

- `limit` (Number) A DNS Zone Record filter option that describe how many records will be percolated.
- `strict` (Boolean) A DNS Zone Record filter option that describe possibility to return answers if no records were percolated through filter.

## Import

Import is supported using the following syntax:

```shell
# import using zone:domain:type format
terraform import edgecenter_dns_zone_record.example_rrset0 example.com:domain.example.com:A
```
