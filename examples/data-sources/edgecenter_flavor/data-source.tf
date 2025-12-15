provider "edgecenter" {
  permanent_api_token = "29422$4ceea35....1513a61c87c68809a4"
}

data "edgecenter_region" "rg" {
  name = "ED-10 Preprod"
}

data "edgecenter_project" "pr" {
  name = "test"
}

data "edgecenter_flavor" "allFlavors" {
  region_id  = data.edgecenter_region.rg.id
  project_id = data.edgecenter_project.pr.id
}


output "allFlavor" {
  value = data.edgecenter_flavor.allFlavors
}

data "edgecenter_flavor" "fullDataFlavors" {
  region_id        = data.edgecenter_region.rg.id
  project_id       = data.edgecenter_project.pr.id
  include_disabled = true
  exclude_windows  = false
  include_prices   = true
}


output "fullDataFlavors" {
  value = data.edgecenter_flavor.fullDataFlavors
}

data "edgecenter_flavor" "bmFlavors" {
  region_id  = data.edgecenter_region.rg.id
  project_id = data.edgecenter_project.pr.id
  type       = "baremetal"
}


output "bmFlavors" {
  value = data.edgecenter_flavor.bmFlavors
}
