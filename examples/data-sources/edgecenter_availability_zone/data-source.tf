provider "edgecenter" {
  permanent_api_token = "29422$4ceea35....1513a61c87c68809a4"
}

data "edgecenter_region" "rg" {
  name = "ED-10 Preprod"
}

data "edgecenter_availability_zone" "region_az" {
  region_id = data.edgecenter_region.rg.id
}

output "availability_zones" {
  value = data.edgecenter_availability_zone.region_az.availability_zones
}