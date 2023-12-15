# Example 1
data "edgecenter_loadbalancer" "lb1" {
  region_id  = var.region_id
  project_id = var.project_id
  name       = "test-loadbalancer"
}

output "lb1" {
  value = data.edgecenter_loadbalancer.lb1
}

# Example 2
data "edgecenter_loadbalancer" "lb2" {
  region_id  = var.region_id
  project_id = var.project_id
  id         = "00000000-0000-0000-0000-000000000000"
}

output "lb2" {
  value = data.edgecenter_loadbalancer.lb2
}
