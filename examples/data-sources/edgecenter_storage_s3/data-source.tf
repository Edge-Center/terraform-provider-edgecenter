provider "edgecenter" {
  permanent_api_token    = "251$d3361.............1b35f26d8"
  edgecenter_storage_api = "https://api.edgecenter.ru/storage"
}

data "edgecenter_storage_s3" "example_s3" {
  name = "example"
}
