provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_storage_sftp" "example_sftp" {
  name       = "example"
  location   = "mia"
  ssh_key_id = [199]
}
