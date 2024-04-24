resource "edgecenter_floatingip" "floating_ip" {
  region_id  = var.region_id
  project_id = var.project_id
}