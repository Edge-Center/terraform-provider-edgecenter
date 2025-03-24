provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

resource "edgecenter_network" "network" {
  name       = "network_example"
  type       = "vxlan"
  region_id  = 1
  project_id = 1
}

resource "edgecenter_subnet" "subnet" {
  name            = "subnet_example"
  cidr            = "192.168.10.0/24"
  network_id      = edgecenter_network.network.id
  dns_nameservers = ["8.8.4.4", "1.1.1.1"]

  enable_dhcp = true

  host_routes {
    destination = "10.0.3.0/24"
    nexthop     = "10.0.0.13"
  }

  host_routes {
    destination = "10.0.4.0/24"
    nexthop     = "10.0.0.14"
  }

  gateway_ip = "192.168.10.1"

  region_id  = 1
  project_id = 1
}