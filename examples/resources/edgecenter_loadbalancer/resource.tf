# Example 1
resource "edgecenter_loadbalancer" "lb" {
  region_id          = var.region_id
  project_id         = var.project_id
  name               = "test-lb"
  flavor_name        = "lb1-1-2"
  vip_network_id     = "00000000-0000-0000-0000-000000000000"
  floating_ip_source = "new"
  metadata = {
    "tag" : "system"
  }
}

# Example 2
# TBD with separate resource
