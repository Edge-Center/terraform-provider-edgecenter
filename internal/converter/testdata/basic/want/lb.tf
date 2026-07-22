resource "edgecenter_loadbalancerv2" "lb" {
  project_id     = 1
  region_id      = 2
  name           = "lb"
  flavor         = "lb1-1-2"
  # TODO(v2migrate): vip_network_id is create time only and ForceNew, V2 does not read it back after import, keep it commented or the plan would replace the load balancer
  # vip_network_id = edgecenter_network.net.id
  # TODO(v2migrate): vip_subnet_id is create time only and ForceNew, V2 does not read it back after import, keep it commented or the plan would replace the load balancer
  # vip_subnet_id  = edgecenter_subnet.sn.id


  metadata_map = {
    env = "dev"
  }
}

resource "edgecenter_lblistener" "lb" {
  project_id = 1
  region_id = 2
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
  name               = "http"
  protocol           = "HTTP"
  protocol_port      = 80
  # TODO(v2migrate): insert_x_forwarded is create time only and is not read back on import, re-adding it right after import would plan a listener replacement
  # insert_x_forwarded = true
  secret_id          = "9b56b91f-4b4c-4a83-9c30-2fda47a0dd54"
}

output "lb_listener_id" {
  value = edgecenter_lblistener.lb.id
}

data "edgecenter_loadbalancerv2" "lb" {
  name       = "lb"
  project_id = 1
  region_id  = 2
}
