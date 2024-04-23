variable "project_id" {
  description = "The ID of the project."
  default = ""
}
variable "region_id" {
  description = "The ID of the region."
}
variable "image_name" {
  description = "The name of the image."
  default = ""
}
variable "permanent_api_token" {
  description = "A permanent API-token."
  default     = ""
}
variable "instance_name" {
  description = "The name of the instance."
  default     = ""
}
variable "instance_volumes" {
  description = "A set defining the volumes to be attached to the instance."
  type = list(object({
    volume_id              = string
    boot_index             = number
    delete_on_termination  = bool
  }))
}
variable "instance_interfaces" {
  description = "A list defining the network interfaces to be attached to the instance."
  type = list(object({
    type                   = string
    network_id             = string
    subnet_id              = string
    port_security_disabled = bool
  }))
}
variable "flavor_id" {
  description = "The ID of the flavor to be used for the instance, determining its compute and memory."
  default     = ""
}
variable "metadata_map" {
  type        = map(string)
  description = "A map containing metadata."
}
variable "server_group" {
  description = "The ID (uuid) of the server group to which the instance should belong."
  default     = ""
}
variable "vm_state" {
  description = "The current virtual machine state of the instance, allowing you to start or stop the VM."
  default     = "active"
}
variable "user_data" {
  description = "A field for specifying user data to be used for configuring the instance at launch time."
  default     = ""
}
variable "username" {
  description = "The username to be used for accessing the instance. Required with password."
  default     = ""
}
variable "password" {
  description = "The password to be used for accessing the instance. Required with username."
  default     = ""
}
variable "keypair_name" {
  description = "The name of the key pair to be associated with the instance for SSH access."
  default     = ""
}