provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_reseller_images" "rimgs" {
  reseller_id = 123456
}

output "view" {
  value = data.edgecenter_reseller_images.rimgs
}


