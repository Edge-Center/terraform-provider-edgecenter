resource "edgecenter_loadbalancer" "lb" {
  region_id  = var.region_id
  project_id = var.project_id
  // other_fields
}

resource "edgecenter_lblistener" "lis" {
  region_id  = var.region_id
  project_id = var.project_id
  // other_fields
}

resource "edgecenter_lbpool" "pool" {
  region_id       = var.region_id
  project_id      = var.project_id
  name            = "test-lbpool"
  lb_algorithm    = "LEAST_CONNECTIONS"
  protocol        = "HTTP"
  loadbalancer_id = edgecenter_loadbalancer.lb.id
  listener_id     = edgecenter_lblistener.lis.id
  healthmonitor {
    type  = "TCP"
    delay = 70
  }
}
