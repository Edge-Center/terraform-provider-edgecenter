resource "edgecenter_reseller_images" "rimgs" {
  reseller_id = 123456

  options {
    region_id = 8
    image_ids = ["b5b4d65d-945f-4b98-ab6f-332319c724ef"]
  }

  options {
    region_id = 9
    image_ids = []
  }
}

data "edgecenter_reseller_images" "cur" {
  reseller_id = 123456
}

output "rimg_entity" {
  value = edgecenter_reseller_images.rimgs.reseller_id
}
