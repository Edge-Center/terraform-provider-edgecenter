resource "edgecenter_loadbalancer" "lb" {
  region_id  = var.region_id
  project_id = var.project_id
  // other_fields
}

resource "edgecenter_lblistener" "listener" {
  region_id          = var.region_id
  project_id         = var.project_id
  name               = "test-lblistener"
  loadbalancer_id    = edgecenter_loadbalancer.lb.id
  protocol_port      = 80
  protocol           = "HTTP"
  insert_x_forwarded = true
  allowed_cidrs      = ["10.10.0.0/24"]
}
