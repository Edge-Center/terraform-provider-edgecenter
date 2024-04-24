data "edgecenter_image" "image" {
  name       = var.image_name
  region_id  = var.region_id
  project_id = var.project_id
}