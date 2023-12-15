# Example 1
data "edgecenter_floatingip" "fip1" {
  region_id           = var.region_id
  project_id          = var.project_id
  floating_ip_address = "10.10.0.1"
}

output "fip1" {
  value = data.edgecenter_floatingip.fip1
}

# Example 2
data "edgecenter_floatingip" "fip2" {
  region_id  = var.region_id
  project_id = var.project_id
  id         = "00000000-0000-0000-0000-000000000000"
}

output "fip2" {
  value = data.edgecenter_floatingip.fip2
}
