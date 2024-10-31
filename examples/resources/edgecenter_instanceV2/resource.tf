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

  host_routes {
    destination = "10.0.3.0/24"
    nexthop     = "10.0.0.13"
  }

  gateway_ip = "192.168.10.1"
  region_id  = 1
  project_id = 1
}

data "edgecenter_image" "ubuntu" {
  name       = "ubuntu-20.04"
  region_id  = 1
  project_id = 1
}

resource "edgecenter_volume" "first_volume" {
  name       = "boot volume"
  type_name  = "ssd_hiiops"
  size       = 5
  region_id  = 1
  project_id = 1
  image_id   = data.edgecenter_image.ubuntu.id
}

resource "edgecenter_volume" "second_volume" {
  name       = "second volume"
  type_name  = "ssd_hiiops"
  size       = 5
  region_id  = 1
  project_id = 1
}

resource "edgecenter_securitygroup" "sg" {
  name       = "example_security_group"
  region_id  = 1
  project_id = 1
  security_group_rules {
    direction      = "egress"
    ethertype      = "IPv4"
    protocol       = "tcp"
    port_range_min = 19990
    port_range_max = 19990
  }
}

resource "edgecenter_instanceV2" "instance" {
  flavor_id = "g1-standard-2-4"
  name      = "test"

  boot_volumes {
    volume_id  = edgecenter_volume.first_volume.id
    boot_index = 0
  }

  data_volumes {
    volume_id = edgecenter_volume.second_volume.id
  }

  interfaces {
    is_default = true
    type       = "subnet"
    network_id = edgecenter_network.network.id
    subnet_id  = edgecenter_subnet.subnet.id
  }

  metadata = {
    some_key = "some_value"
    stage    = "dev"
  }

  configuration {
    key   = "some_key"
    value = "some_data"
  }

  region_id  = 1
  project_id = 1
}

resource "edgecenter_instance_port_security" "port_security" {
  for_each               = { for iface in edgecenter_instanceV2.instance.interfaces : iface.port_id => iface if iface.subnet_id == edgecenter_subnet.subnet.id }
  port_id                = each.key
  instance_id            = edgecenter_instanceV2.instance.id
  region_id              = 1
  project_id             = 1
  port_security_disabled = false
  security_groups {
    overwrite_existing = true
    security_group_ids = [edgecenter_securitygroup.sg.id]
  }
}