resource "edgecenter_servergroup" "default" {
  name       = var.servergroup_name
  policy     = "affinity"
  region_id  = var.region_id
  project_id = var.project_id
}