data "edgecenter_instance" "web" {
  name       = "web"
  project_id = 1
  region_id  = 2
}

output "web_iface_data" {
  value = data.edgecenter_instance.web.interface
}
