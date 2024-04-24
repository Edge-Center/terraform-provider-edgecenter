resource "edgecenter_securitygroup" "sg" {
  for_each     = var.security_groups
  name         = lookup(each.value, "name", null)
  description  = lookup(each.value, "description", null)
  metadata_map = lookup(each.value, "metadata_map", null)


  dynamic "security_group_rules" {
    for_each = each.value.security_group_rules == null ? [] : each.value.security_group_rules
    content {
      direction        = lookup(security_group_rules.value, "direction", null)
      ethertype        = lookup(security_group_rules.value, "ethertype", null)
      protocol         = lookup(security_group_rules.value, "protocol", null)
      port_range_min   = lookup(security_group_rules.value, "port_range_min", null)
      port_range_max   = lookup(security_group_rules.value, "port_range_max", null)
      remote_ip_prefix = lookup(security_group_rules.value, "remote_ip_prefix", null)
    }
  }

  region_id  = var.region_id
  project_id = var.project_id
}