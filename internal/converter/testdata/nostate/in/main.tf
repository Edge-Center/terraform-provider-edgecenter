resource "edgecenter_instance" "solo" {
  flavor_id  = "g1-standard-1-2"
  name       = "solo"
  project_id = 1
  region_id  = 2

  volume {
    source     = "existing-volume"
    volume_id  = "6f81bd26-babf-4083-8b2b-c0e9fb2d6906"
    boot_index = 0
  }

  interface {
    type       = "subnet"
    network_id = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
    subnet_id  = "e7944e55-f957-413d-aa56-0fdca65ebbc7"
  }
}
