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
resource "edgecenter_instance" "web" {
  flavor_id  = "g1-standard-2-4"
  name       = "web"
  project_id = 1
  region_id  = 2

  volume {
    source     = "existing-volume"
    volume_id  = edgecenter_volume.boot.id
    boot_index = 0
  }

  volume {
    source                = "existing-volume"
    volume_id             = edgecenter_volume.data.id
    boot_index            = 1
    delete_on_termination = false
  }

  interface {
    type            = "subnet"
    order           = 1
    network_id      = edgecenter_network.net.id
    subnet_id       = edgecenter_subnet.sn.id
    security_groups = ["d75db0b2-58f1-4a11-88c6-a932bb897310"]
  }

  # public interface
  interface {
    type                   = "external"
    order                  = 0
    port_security_disabled = true
  }

  metadata_map = {
    stage = "dev"
  }

  userdata = "#cloud-config"
}

output "web_id" {
  value = edgecenter_instance.web.id
}

output "web_first_port" {
  value = edgecenter_instance.web.interface.0.port_id
}
