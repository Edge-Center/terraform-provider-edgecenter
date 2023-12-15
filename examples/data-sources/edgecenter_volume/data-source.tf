# Example 1
data "edgecenter_volume" "volume1" {
  region_id  = var.region_id
  project_id = var.project_id
  name       = "test-volume"
}

output "volume1" {
  value = data.edgecenter_volume.volume1
}

# Example 2
data "edgecenter_volume" "volume2" {
  region_id  = var.region_id
  project_id = var.project_id
  id         = "00000000-0000-0000-0000-000000000000"
}

output "volume2" {
  value = data.edgecenter_volume.volume2
}
