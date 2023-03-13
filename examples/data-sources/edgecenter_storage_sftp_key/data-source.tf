provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_storage_sftp_key" "example_key" {
  name = "example"
}
