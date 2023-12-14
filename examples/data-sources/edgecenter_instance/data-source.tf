# Example 1
data "edgecenter_instance" "instance1" {
  region_id  = var.region_id
  project_id = var.project_id
  name       = "test-instance"
}

output "instance1" {
  value = data.edgecenter_instance.instance1
}

# Example 2
data "edgecenter_instance" "instance2" {
  region_id  = var.region_id
  project_id = var.project_id
  id         = "00000000-0000-0000-0000-000000000000"
}

output "instance2" {
  value = data.edgecenter_instance.instance2
}
