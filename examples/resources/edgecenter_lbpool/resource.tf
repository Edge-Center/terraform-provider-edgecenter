provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_loadbalancerv2" "lb" {
  project_id = 1
  region_id  = 1
  name       = "test"
  flavor     = "lb1-1-2"
  metadata_map = {
    tag1 = "tag1_value"
  }
}

resource "edgecenter_lblistener" "listener" {
  project_id      = 1
  region_id       = 1
  name            = "test"
  protocol        = "TCP"
  protocol_port   = 36621
  allowed_cidrs   = ["127.0.0.0/24", "192.168.0.0/24"]
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
}

resource "edgecenter_lbpool" "pl" {
  project_id      = 1
  region_id       = 1
  name            = "test_pool1"
  protocol        = "HTTP"
  lb_algorithm    = "LEAST_CONNECTIONS"
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
  listener_id     = edgecenter_lblistener.listener.id
  health_monitor {
    type        = "PING"
    delay       = 60
    max_retries = 5
    timeout     = 10
  }
  session_persistence {
    type        = "APP_COOKIE"
    cookie_name = "test_new_cookie"
  }
}
