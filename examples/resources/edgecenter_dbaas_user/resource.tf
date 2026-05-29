provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_dbaas_user" "example" {
  cluster_id = "080bbca5-1234-1234-1234-0bccd6f8f1b0"
  project_id = 1
  region_id  = 1
  name       = "my_user"
  password   = "SecurePassword123!"
  databases  = ["my_database"]
}
