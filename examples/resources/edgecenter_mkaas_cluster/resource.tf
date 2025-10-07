provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_mkaas_cluster" "example" {
  name             = "my-cluster01"
  project_id       = 1
  region_id        = 1
  ssh_keypair_name = "tf-keypair"
  network_id       = "6bf878c1-1ce4-47c3-a39b-6b5f1d79bf25"
  subnet_id        = "dc3a3ea9-86ae-47ad-a8e8-79df0ce04839"

  control_plane {
    flavor      = "g3-standard-2-4"
    node_count  = 1
    volume_size = 30
    volume_type = "ssd_hiiops"
    version     = "v1.31.0"
  }
}

