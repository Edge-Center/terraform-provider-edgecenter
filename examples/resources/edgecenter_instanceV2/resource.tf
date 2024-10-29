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

resource "edgecenter_volume" "first_volume" {
  name       = "boot volume"
  type_name  = "ssd_hiiops"
  size       = 5
  image_id   = "f4ce3d30-e29c-4cfd-811f-46f383b6081f"
  region_id  = 1
  project_id = 1
}

resource "edgecenter_volume" "second_volume" {
  name       = "second volume"
  type_name  = "ssd_hiiops"
  size       = 5
  region_id  = 1
  project_id = 1
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

  metadata_map = {
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
  port_id                = [for iface in edgecenter_instanceV2.instance.interfaces : iface.port_id if iface.subnet_id == edgecenter_subnet.subnet1.id][0]
  instance_id            = edgecenter_instanceV2.instance.id
  region_id              = var.region_id
  project_id             = var.project_id
  port_security_disabled = false
  security_groups {
    overwrite_existing = true
    security_group_ids = [edgecenter_securitygroup.sg.id]
  }
}



