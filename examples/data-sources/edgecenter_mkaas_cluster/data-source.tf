provider "edgecenter" {
  permanent_api_token = "251$d3361.............1b35f26d8"
}

locals {
  project_id = 1
  region_id  = 1
}

resource "edgecenter_network" "network" {
  name       = "network_example"
  type       = "vxlan"
  region_id  = local.region_id
  project_id = local.project_id
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

  allocation_pools {
    start = "192.168.10.20"
    end   = "192.168.10.50"
  }

  gateway_ip = "192.168.10.1"

  region_id  = local.region_id
  project_id = local.project_id
}

resource "edgecenter_keypair" "kp" {
  project_id  = local.project_id
  public_key  = "your public key here"
  sshkey_name = "test"
}

resource "edgecenter_mkaas_cluster" "cluster" {
  name                         = "my-cluster01"
  project_id                   = local.project_id
  region_id                    = local.region_id
  ssh_keypair_name             = edgecenter_keypair.kp.sshkey_name
  network_id                   = edgecenter_network.network.id
  subnet_id                    = edgecenter_subnet.subnet.id
  publish_kube_api_to_internet = true

  control_plane {
    flavor      = "g3-standard-2-4"
    node_count  = 1
    volume_size = 30
    volume_type = "ssd_hiiops"
    version     = "v1.31.0"
  }
}

data "edgecenter_mkaas_cluster" "cluster" {
  project_id = local.project_id
  region_id  = local.region_id
  id         = edgecenter_mkaas_cluster.cluster.id

  depends_on = [edgecenter_mkaas_cluster.cluster]
}

output "view" {
  value = data.edgecenter_mkaas_cluster.cluster
}