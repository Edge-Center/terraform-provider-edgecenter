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
  enable_dhcp     = true

  host_routes {
    destination = "10.0.3.0/24"
    nexthop     = "10.0.0.13"
  }

  allocation_pools {
    start = "192.168.10.20"
    end   = "192.168.10.50"
  }

  gateway_ip = "192.168.10.1"

  region_id  = local.region_id
  project_id = local.project_id
}

resource "edgecenter_dbaas_cluster" "cluster" {
  name              = "cluster-example"
  project_id        = local.project_id
  region_id         = local.region_id
  flavor            = "db-g2-standard-2-4-30"
  high_availability = false

  dbms {
    type    = "POSTGRESQL"
    version = "17.5"
  }

  volume {
    volume_size = 30
    volume_type = "db_standard"
  }

  interface {
    network_id = edgecenter_network.network.id
    subnet_id  = edgecenter_subnet.subnet.id
  }
}

data "edgecenter_dbaas_clusters" "cluster" {
  project_id = local.project_id
  region_id  = local.region_id
  name       = edgecenter_dbaas_cluster.cluster.name

  depends_on = [edgecenter_dbaas_cluster.cluster]
}

output "view" {
  value = data.edgecenter_dbaas_clusters.cluster
}
