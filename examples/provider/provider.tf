terraform {
  required_providers {
    edgecenter = {
      source  = "Edge-Center/edgecenter"
      version = ">=1.0.0"
    }
  }
}

provider "edgecenter" {
  api_key              = "251$d3361.............1b35f26d8"
  edgecenter_cloud_api = "https://api.edgecenter.ru/cloud"
}
