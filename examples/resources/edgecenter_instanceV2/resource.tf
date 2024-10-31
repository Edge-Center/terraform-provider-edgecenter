provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

variable "region_id" {
  type        = number
  description = "The region id variable indicates in which region the resource should be created"
  default     = 1
}

variable "project_id" {
  type        = number
  description = "The project id variable specifies in which project the resource should be created"
  default     = 1
}

variable "image_id" {
  type        = string
  description = " The ID of the image to create the volume from. This field is mandatory if creating a volume from an image. Example view 'f4ce3d30-e29c-4cfd-811f-46f383b6081f'."
  default     = "f4ce3d30-e29c-4cfd-811f-46f383b6081f"
}

resource "edgecenter_network" "network" {
  name       = "network_example"
  type       = "vxlan"
  region_id  = var.region_id
  project_id = var.project_id
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
  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_volume" "first_volume" {
  name       = "boot volume"
  type_name  = "ssd_hiiops"
  size       = 5
  region_id  = var.region_id
  project_id = var.project_id
  image_id   = var.image_id
}

resource "edgecenter_volume" "second_volume" {
  name       = "second volume"
  type_name  = "ssd_hiiops"
  size       = 5
  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_securitygroup" "sg" {
  name       = "example_security_group"
  region_id  = var.region_id
  project_id = var.project_id
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

  region_id  = var.region_id
  project_id = var.project_id
}

resource "edgecenter_instance_port_security" "port_security" {
  port_id                = [for iface in edgecenter_instanceV2.instance.interfaces : iface.port_id if iface.subnet_id == edgecenter_subnet.subnet.id][0]
  instance_id            = edgecenter_instanceV2.instance.id
  region_id              = var.region_id
  project_id             = var.project_id
  port_security_disabled = false
  security_groups {
    overwrite_existing = true
    security_group_ids = [edgecenter_securitygroup.sg.id]
  }
}