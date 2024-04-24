output "instance_id" {
  value = edgecenter_instance.instance.id
}
output "flavor_id" {
  value = edgecenter_instance.instance.flavor_id
}
output "instance_volumes" {
  value = edgecenter_instance.instance.volume
}
output "instance_interfaces" {
  value = edgecenter_instance.instance.interface
}
output "vm_state" {
  value = edgecenter_instance.instance.vm_state
}
output "instance_name" {
  value = edgecenter_instance.instance.name
}
output "keypair_name" {
  value = edgecenter_instance.instance.keypair_name
}
output "server_group" {
  value = edgecenter_instance.instance.server_group
}
output "user_data" {
  value = edgecenter_instance.instance.user_data
}
output "metadata_map" {
  value = edgecenter_instance.instance.metadata_map
}