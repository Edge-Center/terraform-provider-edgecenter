provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_mkaas_pool" "apps" {
  project_id = 1234
  region_id  = 53
  cluster_id = 321
  pool_id    = 654
}


