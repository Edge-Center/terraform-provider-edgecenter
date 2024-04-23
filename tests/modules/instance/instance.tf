resource "edgecenter_instance" "instance" {
  flavor_id    = var.flavor_id
  name         = var.instance_name
  server_group = var.server_group
  vm_state     = var.vm_state
  user_data    = var.user_data
  username     = var.username
  password     = var.password
  keypair_name = var.keypair_name
  metadata_map = var.metadata_map

  dynamic "volume" {
    for_each = var.instance_volumes
    content {
      source                = "existing-volume"
      volume_id             = lookup(volume.value, "volume_id", null)
      boot_index            = lookup(volume.value, "boot_index", null)
      delete_on_termination = lookup(volume.value, "delete_on_termination",null)
    }
  }

  dynamic "interface" {
    for_each = var.instance_interfaces
    content {
      type                   = interface.value["type"]
      fip_source             = lookup(interface.value, "fip_source", null)
      ip_address             = lookup(interface.value, "ip_address", null)
      network_id             = lookup(interface.value, "network_id", null)
      order                  = lookup(interface.value, "order", null)
      port_id                = lookup(interface.value, "port_id", null)
      port_security_disabled = lookup(interface.value, "port_security_disabled", null)
      subnet_id              = lookup(interface.value, "subnet_id", null)
      security_groups        = lookup(interface.value, "security_groups", null)
    }
  }

  region_id  = var.region_id
  project_id = var.project_id
}



