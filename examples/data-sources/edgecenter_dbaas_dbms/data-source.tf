provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_dbaas_dbms" "example" {
  project_id = 1
  region_id  = 1
}
