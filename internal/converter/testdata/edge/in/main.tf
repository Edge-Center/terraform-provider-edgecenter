variable "volume_ids" {
  type = list(string)
}

variable "boot_index" {
  type = number
}

resource "edgecenter_instance" "workers" {
  count      = 2
  flavor_id  = "g1-standard-1-2"
  name       = "w-${count.index}"
  project_id = 1
  region_id  = 2

  volume {
    source     = "existing-volume"
    volume_id  = var.volume_ids[count.index]
    boot_index = var.boot_index
  }

  interface {
    type       = "any_subnet"
    network_id = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
  }

  metadata {
    key   = "env"
    value = "dev"
  }
}

resource "edgecenter_instance" "min" {
  flavor_id  = "g1-standard-1-2"
  name       = "min"
  project_id = 1
  region_id  = 2

  volume {
    source    = "existing-volume"
    volume_id = "6f81bd26-babf-4083-8b2b-c0e9fb2d6906"
  }

  interface {
    network_id = "faf6507b-1ff1-4ebf-b540-befd5c09fe06"
    subnet_id  = "e7944e55-f957-413d-aa56-0fdca65ebbc7"
  }
}
