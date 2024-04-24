resource "edgecenter_reservedfixedip" "fixed_ip" {
  for_each                      = var.reserved_fixed_ips
  type                          = lookup(each.value, "type", null)
  instance_ports_that_share_vip = lookup(each.value, "instance_ports_that_share_vip", null)
  is_vip                        = lookup(each.value, "is_vip", null)
  network_id                    = lookup(each.value, "network_id", null)
  subnet_id                     = lookup(each.value, "subnet_id", null)
  fixed_ip_address              = lookup(each.value, "fixed_ip_address", null)

  dynamic "allowed_address_pairs" {
    for_each = each.value.allowed_address_pairs == null ? [] : each.value.allowed_address_pairs
    content {
      ip_address = lookup(allowed_address_pairs.value,"ip_address",null)
      mac_address = lookup(allowed_address_pairs.value,"mac_address",null)
    }
  }

  region_id  = var.region_id
  project_id = var.project_id
}