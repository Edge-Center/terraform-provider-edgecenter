resource "edgecenter_loadbalancer" "lb" {
  project_id     = 1
  region_id      = 2
  name           = "lb"
  flavor         = "lb1-1-2"
  vip_network_id = edgecenter_network.net.id
  vip_subnet_id  = edgecenter_subnet.sn.id

  listener {
    name               = "http"
    protocol           = "HTTP"
    protocol_port      = 80
    insert_x_forwarded = true
    secret_id          = "9b56b91f-4b4c-4a83-9c30-2fda47a0dd54"
  }

  metadata_map = {
    env = "dev"
  }
}

output "lb_listener_id" {
  value = edgecenter_loadbalancer.lb.listener.0.id
}

data "edgecenter_loadbalancer" "lb" {
  name       = "lb"
  project_id = 1
  region_id  = 2
}
