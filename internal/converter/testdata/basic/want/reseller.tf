resource "edgecenter_reseller_imagesV2" "rimgs" {
  entity_id = 123456

  options {
    region_id = 8
    image_ids = ["b5b4d65d-945f-4b98-ab6f-332319c724ef"]
  }

  options {
    region_id = 9
    image_ids = []
  }
  entity_type = "reseller"
}

data "edgecenter_reseller_imagesV2" "cur" {
  entity_id = 123456
  entity_type = "reseller"
}

output "rimg_entity" {
  value = edgecenter_reseller_imagesV2.rimgs.entity_id
}
