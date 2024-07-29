provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_project" "pr" {
  name = "test"
}

data "edgecenter_region" "rg" {
  name = "ED-10 Preprod"
}

data "edgecenter_instance_port_security" "port_security" {
  region_id   = data.edgecenter_region.rg.id
  project_id  = data.edgecenter_project.pr.id
  port_id     = "073947f8-8589-4104-bdff-2cedbe56239f"
  instance_id = "4f81e8f8-d7b8-45a4-93fd-609ad2n670f0"
}

output "view" {
  value = data.edgecenter_instance_port_security.port_security
}

