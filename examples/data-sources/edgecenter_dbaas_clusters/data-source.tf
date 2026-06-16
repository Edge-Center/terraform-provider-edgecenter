provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_dbaas_clusters" "example" {
  project_id = 1
  region_id  = 1
  name       = "cluster-example"
}

output "view" {
  value = data.edgecenter_dbaas_clusters.example
}
