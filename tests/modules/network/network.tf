resource "edgecenter_network" "network" {
  name       = var.network_name
  type       = "vxlan"
  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_subnet" "subnet" {
  name            = var.subnet_name
  cidr            = "192.168.10.0/24"
  network_id      = edgecenter_network.network.id
  dns_nameservers = ["8.8.4.4", "1.1.1.1"]

  gateway_ip = "192.168.10.1"
  region_id  = var.region_id
  project_id = var.project_id
}