provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_dbaas_cluster" "example" {
  name              = "cluster-example"
  project_id        = 1
  region_id         = 1
  flavor            = "db-g2-standard-2-4-30"
  high_availability = false

  dbms {
    type    = "POSTGRESQL"
    version = "17.5"
  }

  volume {
    volume_size = 30
    volume_type = "db_standard"
  }

  interface {
    network_id = "6bf878c1-1ce4-47c3-a39b-6b5f1d79bf25"
    subnet_id  = "dc3a3ea9-86ae-47ad-a8e8-79df0ce04839"
  }
}
