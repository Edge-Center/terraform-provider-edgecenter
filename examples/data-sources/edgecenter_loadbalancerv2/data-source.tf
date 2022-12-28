provider edgecenter {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

data "edgecenter_project" "pr" {
  name = "test"
}

data "edgecenter_region" "rg" {
  name = "ED-10 Preprod"
}

data "edgecenter_loadbalancerv2" "lb" {
  name       = "test-lb"
  region_id  = data.edgecenter_region.rg.id
  project_id = data.edgecenter_project.pr.id
}

output "view" {
  value = data.edgecenter_loadbalancerv2.lb
}

