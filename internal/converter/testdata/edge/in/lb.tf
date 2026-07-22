resource "edgecenter_loadbalancer" "farm" {
  for_each   = { a = "lb-a", b = "lb-b" }
  project_id = 1
  region_id  = 2
  name       = each.value
  flavor     = "lb1-1-2"

  listener {
    name          = "l"
    protocol      = "TCP"
    protocol_port = 8080
  }
}
