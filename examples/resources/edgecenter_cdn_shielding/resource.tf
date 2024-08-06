provider "edgecenter" {
  permanent_api_token = "29422$4ceea35....1513a61c87c68809a4"
}

resource "edgecenter_cdn_origingroup" "source_group_1" {
  name     = "Source group 1"
  use_next = true
  origin {
    source  = "example.com"
    enabled = true
  }
}

resource "edgecenter_cdn_resource" "cdn_res_1" {
  cname        = "cdn.example.com"
  origin_group = edgecenter_cdn_origingroup.source_group_1.id
}

data "edgecenter_cdn_shielding_location" "shield_dc" {
  datacenter = "dt"
}

resource "edgecenter_cdn_shielding" "shielding" {
  resource_id   = edgecenter_cdn_resource.cdn_res_1.id
  shielding_pop = data.edgecenter_cdn_shielding_location.shield_dc.id
}
