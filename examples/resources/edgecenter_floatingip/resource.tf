resource "edgecenter_floatingip" "fip" {
  region_id  = var.region_id
  project_id = var.project_id
  metadata = {
    "key1" : "value1",
    "key2" : "value2",
  }
}
