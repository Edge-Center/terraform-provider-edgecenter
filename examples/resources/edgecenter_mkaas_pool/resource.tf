provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_mkaas_pool" "apps" {
  cluster_id = 53

  # Core pool parameters
  name        = "apps-pool"
  flavor      = "mkaas-worker-g3-cpu-2-2"
  volume_size = 20
  volume_type = "standard"

  # Optional fields
  # security_group_ids = ["b4a1b1d3-xxxx-xxxx-xxxx-1b2c3d4e5f6a"]
  # labels = {
  #   key = "val"
  # }

  taints {
    key    = "dedicated"
    value  = "gpu"
    effect = "NoSchedule"
  }

  # Manual node count management
  node_count = 3

  # The presence of the `auto_scale` block enables the autoscaler; remove the block to disable it.
  # scale_policy {
  #   auto_scale {
  #     min_node_count = 1
  #     max_node_count = 5
  #   }
  # }

  project_id = 1234
  region_id  = "1234"
}
