provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_reseller_imagesV2" "rimgs" {
  entity_id   = 123456
  entity_type = "reseller"
}

output "view" {
  value = data.edgecenter_reseller_imagesV2.rimgs
}


