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

  interface {
    type                   = "subnet"
    network_id             = edgecenter_network.network.id
    subnet_id              = edgecenter_subnet.subnet.id
    security_groups        = ["d75db0b2-58f1-4a11-88c6-a932bb897310"]
    port_security_disabled = true
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

//***
// another one example with one interface to private network and fip to internet
//***

resource "edgecenter_reservedfixedip" "fixed_ip" {
  project_id       = 1
  region_id        = 1
  type             = "ip_address"
  network_id       = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
  fixed_ip_address = "192.168.13.6"
  is_vip           = false
}

resource "edgecenter_volume" "first_volume" {
  name       = "boot volume"
  type_name  = "ssd_hiiops"
  size       = 10
  image_id   = "6dc4e061-6fab-41f3-91a3-0ba848fb32d9"
  project_id = 1
  region_id  = 1
}

resource "edgecenter_floatingip" "fip" {
  project_id       = 1
  region_id        = 1
  fixed_ip_address = edgecenter_reservedfixedip.fixed_ip.fixed_ip_address
  port_id          = edgecenter_reservedfixedip.fixed_ip.port_id
}


resource "edgecenter_instanceV2" "v" {
  project_id = 1
  region_id  = 1
  name       = "hello"
  flavor_id  = "g1-standard-1-2"

  boot_volumes {
    volume_id  = edgecenter_volume.first_volume.id
    boot_index = 0
  }

  interface {
    type            = "reserved_fixed_ip"
    port_id         = edgecenter_reservedfixedip.fixed_ip.port_id
    fip_source      = "existing"
    existing_fip_id = edgecenter_floatingip.fip.id
    security_groups = ["ada84751-fcca-4491-9249-2dfceb321616"]
  }
}



