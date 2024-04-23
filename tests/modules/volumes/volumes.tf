resource "edgecenter_volume" "first_volume" {
  name       = "boot volume"
  type_name  = "standard"
  size       = 30
  image_id   = data.edgecenter_image.image.id
  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_volume" "second_volume" {
  name       = "second volume"
  type_name  = "standard"
  size       = 5
  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_volume" "third_volume" {
  name       = "third volume"
  type_name  = "standard"
  size       = 5
  region_id  = var.region_id
  project_id = var.project_id
}