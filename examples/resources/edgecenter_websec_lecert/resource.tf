provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_websec_lecert" "example_com" {
  resource_id = 12345
}
