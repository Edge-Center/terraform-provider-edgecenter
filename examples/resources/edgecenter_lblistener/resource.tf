provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_loadbalancerv2" "lb" {
  project_id = 1
  region_id  = 1
  name       = "test"
  flavor     = "lb1-1-2"
}

resource "edgecenter_lblistener" "listener" {
  project_id      = 1
  region_id       = 1
  name            = "test"
  protocol        = "TCP"
  protocol_port   = 36621
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
}