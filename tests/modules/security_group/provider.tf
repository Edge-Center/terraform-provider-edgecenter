terraform {
  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
    }
  }
}
provider "edgecenter" {
  permanent_api_token = var.permanent_api_token
}