provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_k8s_pool" "pool" {
  project_id = 1
  region_id  = 1
  cluster_id = "6bf878c1-1ce4-47c3-a39b-6b5f1d79bf25"
  pool_id    = "dc3a3ea9-86ae-47ad-a8e8-79df0ce04839"
}

