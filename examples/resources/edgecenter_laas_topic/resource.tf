provider edgecenter {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_laas_topic" "test" {
  region_id  = 1
  project_id = 1

  name              = "test"
}
