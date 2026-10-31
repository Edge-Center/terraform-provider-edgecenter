---
page_title: "Moving from edgecenter_protection_* to edgecenter_websec_*"
subcategory: ""
description: |-
  How to bring domains moved to the WebSec API under edgecenter_websec_* resources with removed and import blocks, without recreating anything.
---

# Moving from edgecenter_protection_* to edgecenter_websec_*

The `edgecenter_protection_*` resources work with the old protection API. EdgeCenter moves the data of
a client from it to the WebSec API, and from then on the domains are managed with the
`edgecenter_websec_*` resources. The configuration is not converted automatically. The old resources
leave the state through `removed` blocks, the new ones come in through `import` blocks, and
`terraform plan -generate-config-out` writes their configuration from what the WebSec API holds after
the move. A mechanical translation of the old configuration would not match it: the move adds TLS
versions and network masks and drops some options, so the plan would show changes where the goal is
only to import.

`removed` blocks need Terraform 1.7 or later. Do not manage a domain with an old and a new resource at
the same time: changes made through one of them show up as drift in the other.

The old resources keep working after the move, because the WebSec API answers the requests of the old
API too. The one difference is `multiple_origins` and `wildcard_aliases` of
`edgecenter_protection_resource`: they come from the permissions of the client now, and the value in
the configuration is ignored. A value that differs from the permission shows up as a change in every
plan. Delete both attributes from the configuration to stop it.

## Where the data goes

The move keeps the ids of resources, certificates, aliases (now subdomains) and named options (now
locations). Source groups are created anew, one for every resource that had an origin, named
`Default group for <domain>`. Origins, headers and list entries have no ids of their own any more:
they become parts of the source group and of the resource options and come in together with them.

| Old resource | New place | Import id |
|---|---|---|
| `edgecenter_protection_resource` (`N`) | `edgecenter_websec_resource` | `N` |
| `edgecenter_protection_resource_origin` (`N:O`) | a `source` block of `edgecenter_websec_source_group`, `source_group` of the resource | `resource:N` for the group |
| `edgecenter_protection_resource_header` (`N:H`) | an element of `options.response_headers.value`, `name = key` and `value = [value]` | comes with the resource |
| `edgecenter_protection_resource_blacklist_entry`, `edgecenter_protection_resource_whitelist_entry` (`N:E`) | an element of `options.blacklist.value` or `options.whitelist.value` | comes with the resource |
| `edgecenter_protection_resource_alias` (`N:A`) | an element of `subdomains` | comes with the resource |
| `edgecenter_protection_resource_certificate` (`N`) with `ssl_type = "custom"` | `edgecenter_websec_ssl_certificate` and `ssl_certificate` of the resource | `resource:N` |
| `edgecenter_protection_resource_certificate` (`N`) with `ssl_type = "le"` | `edgecenter_websec_lecert`; the resource has no `ssl_certificate` in the configuration | `N` |
| `edgecenter_protection_resource_alias_certificate` (`N:A`) | nothing: the WebSec API has no certificate of a subdomain | `removed` only |

The attributes of `edgecenter_protection_resource` move like this:

- `name` becomes `domain`, `active` stays `active`;
- `tls` (`1`, `1.1`, `1.2`, `1.3`) becomes `options.tls_versions` (`TLSv1`, `TLSv1.1`, `TLSv1.2`, `TLSv1.3`), and the move adds `TLSv1.2` and `TLSv1.3` where they were missing;
- `geoip_mode` and `geoip_list` become `options.country_acl`: `allow` turns into `policy_type = "allow"`, `block` into `policy_type = "deny"`, the countries are in uppercase, and with `no` or an empty list there is no option at all;
- `http_to_origin` becomes `options.source_protocol_https` with the opposite value;
- `load_balancing_type` becomes `options.ip_hash`, where `Round Robin with session persistence` is `true`;
- `redirect_to_https` becomes `options.redirect_http_to_https`, which stays enabled only when the resource had a certificate;
- `waf` becomes `options.waf`, `www_redirect` becomes `options.www_redirects`;
- `client`, `enabled`, `ip` and `status` become `client`, `enabled`, `protected_ip` and `status`, and `wait_for_le` is gone;
- `multiple_origins` and `wildcard_aliases` are permissions of the client now, and only an administrator changes them.

## Before you start

Check that the data is in the WebSec API already. The provider setting `edgecenter_websec_api` points
at it, by default `<api_endpoint>/v2/protection`, and the paths below are relative to it.
`GET /resources/N` must answer 200 for every old resource, and `protected_ip` in the answer must be the
`ip` attribute of the old `edgecenter_protection_resource`. If `protected_ip` is `null` or another
address, stop and contact support. The first update of such a resource would take a new address from
the pool while the A record of the domain points at the old one. The provider refuses to update a
resource without `protected_ip`, but import does not show the problem.

## Moving

First take the old resources out of the configuration and put a `removed` block with
`destroy = false` in place of each of them. A resource with `count` or `for_each` needs one `removed`
block for all its instances.

```terraform
removed {
  from = edgecenter_protection_resource.site

  lifecycle {
    destroy = false
  }
}

removed {
  from = edgecenter_protection_resource_origin.site

  lifecycle {
    destroy = false
  }
}

removed {
  from = edgecenter_protection_resource_certificate.site

  lifecycle {
    destroy = false
  }
}
```

If the old API no longer answers for the client, plan fails on the refresh of the old resources. Take
them out of the state with `terraform state rm` instead of `removed`:

```shell
terraform state list | grep edgecenter_protection_ | while read -r address; do terraform state rm "$address"; done
```

Then add an `import` block for every new object: the resource by `N`, its source group by
`resource:N`, its own certificate by `resource:N`. The group and the certificate can be found by the id
of the old resource, so nothing has to be looked up in the API.

```terraform
import {
  to = edgecenter_websec_resource.site
  id = "12345"
}

import {
  to = edgecenter_websec_source_group.site
  id = "resource:12345"
}

import {
  to = edgecenter_websec_ssl_certificate.site
  id = "resource:12345"
}
```

A Let's Encrypt certificate is imported as `edgecenter_websec_lecert` by the resource id instead:

```terraform
import {
  to = edgecenter_websec_lecert.site
  id = "12345"
}
```

Locations made from the named options of the old API are imported by `N:L` if Terraform should manage
them; `GET /resources/N/locations` lists them. Their `location` is the name of the old named option. If
it does not start with `/` or `^/`, neither plan nor the API accepts the location, and it has to get a
proper pattern.

Generate the configuration:

```shell
terraform plan -generate-config-out=websec.tf
```

Edit `websec.tf` before the next plan:

- replace the number in `source_group` with a reference to the imported group, and the number in `ssl_certificate` with a reference to the imported certificate; a resource whose certificate comes from `edgecenter_websec_lecert` must not have `ssl_certificate` at all;
- delete `ssl_certificate = 0` from resources without a certificate and `source_group = 0` from locations without a group of their own: 0 is how the state shows a missing reference, and plan rejects it;
- set `certificate` and `private_key` of each certificate from `ssl_crt` and `ssl_key` of the old `edgecenter_protection_resource_certificate`, because the API does not return them;
- fix what the next section lists.

Run `terraform plan` again. It should show the imports, an in-place update of every certificate, which
only writes the certificate and the key to the state and sends nothing to the API, and the changes you
made on purpose. Before applying, compare `cert_subject_cn` and `validity_not_after` of every imported
certificate in the plan with the output of `openssl x509 -noout -subject -enddate` for the PEM you put
in. Then apply.

## What has to be fixed by hand

Headers with the same name arrive as separate elements of `options.response_headers.value`, but the
API requires unique names. Merge them into one header with a list of values, or the first update that
sends the headers fails.

The old resource sent `max_fails` of an origin only when it was set, and the old API took 0 then. The
WebSec API accepts `max_fails` from 1, so a generated configuration with `max_fails = 0` does not pass
plan. The two values behave differently: with 0 an origin is never marked unavailable, while with 1 a
primary origin drops out for `fail_timeout` seconds after its first failure and the traffic goes to a
backup. Choose `max_fails` and `fail_timeout` deliberately. The default `fail_timeout` of the WebSec
API is 1 second; the old one was 10.

An origin without a mode comes out as `mode = ""`, which plan rejects. Set `primary`, `backup` or
`down`.

A source group in which the weights of the primary sources add up to more than 100 is moved as it is,
but plan rejects it and the API accepts no change to it either. Fix the weights. A resource that had no
live origin has no group at all and needs a new `edgecenter_websec_source_group`.

Subdomains named like `<timestamp>_<id>_<alias>` are aliases that had been deleted in the old API.
Remove them from `subdomains`, and do it before adding `edgecenter_websec_lecert`: plan rejects such
names, the API accepts no update of the resource while they are there, and a Let's Encrypt request
fails on them.

If import of a certificate by `resource:N` fails because the resource refers to a deleted certificate,
contact support: the API fails every update of such a resource. The uploaded certificate itself is in
the WebSec API under an id of its own. Find it with `GET /ssl/certificates?search=<name>` and import it
by that id.

`edgecenter_protection_resource_alias_certificate` has no counterpart, so it only gets a `removed`
block. The certificate itself is in the WebSec API and stays attached to its subdomain, but do not
import it into `edgecenter_websec_ssl_certificate`. Its `has_related_resources` does not see
subdomains, and deleting or replacing it deletes the subdomain. The API cannot change the certificate
of a subdomain: to get rid of it, remove the subdomain and add it again.

Addresses in `blacklist` and `whitelist` are stored as networks, `9.9.9.9` as `9.9.9.9/32`. The
provider treats both forms as the same value, so either can stay in the configuration.

A moved resource keeps the `status` string of the old API. The server renews a Let's Encrypt
certificate only for a resource whose `status` is `active`, so check `status` of the resources that use
`edgecenter_websec_lecert` and contact support if it is something else.
