provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_project" "project_resource_name" {
  name = "test"
  description = "test description"
}
