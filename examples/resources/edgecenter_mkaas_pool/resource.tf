provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_mkaas_pool" "apps" {
  cluster_id = 53

  # Основные параметры пула
  name        = "apps-pool"
  flavor      = "g3-standard-2-4"
  node_count  = 3
  volume_size = 20
  volume_type = "standard"

  #   # Необязательные поля
  #   security_group_id = "b4a1b1d3-xxxx-xxxx-xxxx-1b2c3d4e5f6a"


  project_id = 1234
  region_id  = "1234"
}
