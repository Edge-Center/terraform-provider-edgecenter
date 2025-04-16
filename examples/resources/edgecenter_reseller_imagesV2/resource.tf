provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_reseller_imagesV2" "rimgs" {
  entity_id   = 123456
  entity_type = "reseller"
  options {
    region_id = 123
    image_ids = ["b5b4d65d-945f-4b98-ab6f-332319c724ef", "0052a312-e6d8-4177-8e29-b017a3a6b588"]
  }
  options {
    region_id = 456
    image_ids = []
  }
}