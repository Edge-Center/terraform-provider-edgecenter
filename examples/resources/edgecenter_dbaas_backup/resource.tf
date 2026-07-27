provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_dbaas_backup" "example" {
  name        = "backup-example"
  project_id  = 1
  region_id   = 1
  cluster_id  = "080bbca5-1234-1234-1234-0bccd6f8f1b0"
  description = "Manual backup of the cluster"
}
