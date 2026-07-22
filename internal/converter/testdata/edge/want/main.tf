variable "volume_ids" {
  type = list(string)
}

variable "boot_index" {
  type = number
}

resource "edgecenter_instanceV2" "workers" {
  count      = 2
  flavor_id  = "g1-standard-1-2"
  name       = "w-${count.index}"
  project_id = 1
  region_id  = 2

  # TODO(v2migrate): boot_index must be a literal number to classify this volume, move the block to boot_volumes or data_volumes manually
  # volume {
    # source     = "existing-volume"
    # volume_id  = var.volume_ids[count.index]
    # boot_index = var.boot_index
  # }

  interfaces {
    # TODO(v2migrate): any_subnet is not supported in V2, use subnet with an explicit subnet_id
    # type       = "any_subnet"
    network_id = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
    is_default = true
  }

  # TODO(v2migrate): deprecated metadata blocks are gone in V2, move the entries into the metadata map attribute
  # metadata {
    # key   = "env"
    # value = "dev"
  # }
}

resource "edgecenter_instanceV2" "min" {
  flavor_id  = "g1-standard-1-2"
  name       = "min"
  project_id = 1
  region_id  = 2

  boot_volumes {
    volume_id = "6f81bd26-babf-4083-8b2b-c0e9fb2d6906"
    boot_index = 0
  }

  interfaces {
    network_id = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
    subnet_id  = "e7944e55-f957-413d-aa56-0fdca65ebbc7"
    # TODO(v2migrate): type is required in V2, set subnet, external or reserved_fixed_ip
    is_default = true
  }
}
