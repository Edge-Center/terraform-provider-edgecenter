terraform {
#   required_version = ">= 0.1.22"

  required_providers {
    edgecenter = {
      source = "local.edgecenter.ru/repo/edgecenter"
      version = "0.1.22"  # need to specify
    }
  }
}

provider "edgecenter" {
  permanent_api_token = "5623$84777d4986958985850cf76d81322219e6b051ca4825f72aae49d1aba1e2e8e56677bdd4b1c02f2a6bc3d77df9d2808bbc377070cf249d6c726683f8e8104ede"
}

resource "edgecenter_dns_zone" "terraform-zone" {
  name = "terraform-zone.com"
}
