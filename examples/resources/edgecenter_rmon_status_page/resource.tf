provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_rmon_status_page" "example" {
  name = "Example Status"
  slug = "example-status"

  checks {
    check_id = 12345
  }
}
