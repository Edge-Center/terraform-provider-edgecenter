provider "edgecenter" {
  permanent_api_token = "token"
}

resource "edgecenter_network" "net" {
  name       = "net"
  type       = "vxlan"
  region_id  = 2
  project_id = 1
}

resource "edgecenter_subnet" "sn" {
  name       = "sn"
  cidr       = "192.168.10.0/24"
  network_id = edgecenter_network.net.id
  region_id  = 2
  project_id = 1
}

resource "edgecenter_volume" "boot" {
  name       = "boot"
  type_name  = "ssd_hiiops"
  size       = 10
  image_id   = "f4ce3d30-e29c-4cfd-811f-46f383b6081f"
  region_id  = 2
  project_id = 1
}

resource "edgecenter_volume" "data" {
  name       = "data"
  type_name  = "standard"
  size       = 5
  region_id  = 2
  project_id = 1
}

# web server
resource "edgecenter_instanceV2" "web" {
  flavor_id  = "g1-standard-2-4"
  name       = "web"
  project_id = 1
  region_id  = 2

  boot_volumes {
    volume_id  = edgecenter_volume.boot.id
    boot_index = 0
  }

  data_volumes {
    volume_id             = edgecenter_volume.data.id
    # TODO(v2migrate): delete_on_termination is not supported in V2, volume deletion is controlled by the edgecenter_volume resource
    # delete_on_termination = false
  }

  interfaces {
    type            = "subnet"
    network_id      = edgecenter_network.net.id
    subnet_id       = edgecenter_subnet.sn.id
    # TODO(v2migrate): interface security_groups are gone in V2, manage them with the edgecenter_instance_port_security resource
    # security_groups = ["d75db0b2-58f1-4a11-88c6-a932bb897310"]
  }

  # public interface
  interfaces {
    type                   = "external"
    # TODO(v2migrate): port_security_disabled is gone in V2, manage it with the edgecenter_instance_port_security resource
    # port_security_disabled = true
    is_default = true
  }

  metadata = {
    stage = "dev"
  }

  user_data = "#cloud-config"
}

output "web_id" {
  value = edgecenter_instanceV2.web.id
}

output "web_first_port" {
  # TODO(v2migrate): interface: interface outputs moved to the interfaces set in V2, sets are not index addressable
  value = edgecenter_instanceV2.web.interface.0.port_id
}
