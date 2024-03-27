provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_loadbalancerv2" "lb" {
  flavor       = "lb1-1-2"
  metadata_map = {}
  name         = "test-l7policy"
  project_id   = 1
  region_id    = 1
}

resource "edgecenter_lblistener" "listener" {
  project_id      = 1
  region_id       = 1
  name            = "test-l7policy"
  protocol        = "HTTP"
  protocol_port   = 80
  loadbalancer_id = edgecenter_loadbalancerv2.lb.id
}


resource "edgecenter_lb_l7policy" "l7policy" {
  name               = "test-policy"
  project_id         = 1
  region_id          = 1
  action             = "REDIRECT_PREFIX"
  listener_id        = edgecenter_lblistener.listener.id
  redirect_http_code = 303
  redirect_prefix    = "https://your-prefix.com/"
  tags               = ["aaa", "bbb", "ccc"]
}


