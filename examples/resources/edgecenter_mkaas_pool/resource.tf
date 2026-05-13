provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_mkaas_pool" "apps" {
  cluster_id = 53

  # Основные параметры пула
  name        = "apps-pool"
  flavor      = "mkaas-worker-g3-cpu-2-2"
  volume_size = 20
  volume_type = "standard"

  # Необязательные поля
  # security_group_ids = ["b4a1b1d3-xxxx-xxxx-xxxx-1b2c3d4e5f6a"]
  # labels = {
  #   key = "val"
  # }

  taints {
    key    = "dedicated"
    value  = "gpu"
    effect = "NoSchedule"
  }

  # Ручное управление количеством узлов
  node_count = 3

  # Наличие блока `auto_scale` включает автоскейлер; удалите блок, чтобы отключить.
  # scale_policy {
  #   auto_scale {
  #     min = 1
  #     max = 5
  #   }
  # }

  project_id = 1234
  region_id  = "1234"
}
