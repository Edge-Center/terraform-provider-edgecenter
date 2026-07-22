resource "edgecenter_loadbalancerv2" "farm" {
  for_each   = { a = "lb-a", b = "lb-b" }
  project_id = 1
  region_id  = 2
  name       = each.value
  flavor     = "lb1-1-2"

  # TODO(v2migrate): the nested listener moved to a standalone edgecenter_lblistener resource, cannot extract automatically when the parent uses count or for_each, create the edgecenter_lblistener resource manually
  # listener {
    # name          = "l"
    # protocol      = "TCP"
    # protocol_port = 8080
  # }
}
