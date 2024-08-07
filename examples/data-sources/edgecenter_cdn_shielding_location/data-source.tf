provider "edgecenter" {
  permanent_api_token = "29422$4ceea35....1513a61c87c68809a4"
}

data "edgecenter_cdn_shielding_location" "shield_dc" {
  datacenter = "dt"
}

resource "edgecenter_cdn_shielding" "shielding" {
  resource_id   = 1
  shielding_pop = data.edgecenter_cdn_shielding_location.shield_dc.id
}
