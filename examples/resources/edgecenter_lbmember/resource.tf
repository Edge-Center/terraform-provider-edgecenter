resource "edgecenter_lbpool" "pool" {
  region_id  = var.region_id
  project_id = var.project_id
  // other_fields
}

resource "edgecenter_lbmember" "member" {
  region_id     = var.region_id
  project_id    = var.project_id
  pool_id       = edgecenter_lbpool.pool.id
  address       = "10.10.0.7"
  protocol_port = 9099
  weight        = 20
}
