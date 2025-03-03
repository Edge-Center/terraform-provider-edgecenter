provider "edgecenter" {
  # This data source has been created for resellers and only works with the reseller API key.
  permanent_api_token = "251$d3361.............1b35f26d8"
}


data "edgecenter_reseller_networks" "rnw" {
  shared   = false
  order_by = "name.desc"
  metadata_kv = {
    key_1 = "value_1"
  }

  metadata_k = ["key_1"]
}

output "view" {
  value = data.edgecenter_reseller_networks.rnw
}

