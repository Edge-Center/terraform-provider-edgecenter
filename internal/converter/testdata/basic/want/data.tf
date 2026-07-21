data "edgecenter_instanceV2" "web" {
  name       = "web"
  project_id = 1
  region_id  = 2
}

output "web_iface_data" {
  # TODO(v2migrate): interface: interface outputs are exposed as the interfaces list in the V2 data source
  value = data.edgecenter_instanceV2.web.interface
}
