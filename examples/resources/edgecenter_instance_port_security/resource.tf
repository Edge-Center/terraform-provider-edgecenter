provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_instance_port_security" "port_security" {
  port_id                = "073947f8-8589-4104-bdff-2cedbe56239f"
  instance_id            = "4f81e8f8-d7b8-45a4-93fd-609ad2n670f0"
  region_id              = 1
  project_id             = 1
  port_security_disabled = false
  security_groups {
    overwrite_existing = true
    security_group_ids = ["cd114905-1bc7-45d7-9def-463f16379563", "4c2fb2a4-8535-474e-aa7f-ac35804de389"]
  }
}



