output "security_group_ids" {
  value = [ for k,v in var.security_groups: edgecenter_securitygroup.sg[k].id ]
}