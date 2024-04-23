variable "project_id" {
  description = "The ID of the project."
  default = ""
}
variable "region_id" {
  description = "The ID of the region."
}
variable "permanent_api_token" {
  description = "A permanent API-token."
  default     = ""
}
variable "reserved_fixed_ips" {
  description = "A reserved fixed IP is an IP address within a specific network that is reserved for a particular purpose. Reserved fixed IPs are typically not automatically assigned to instances but are instead set aside for specific needs or configurations"
  type = map(object({
    type                          = string
    instance_ports_that_share_vip = optional(list(string))
    is_vip                        = optional(bool)
    network_id                    = optional(string)
    subnet_id                     = optional(string)
    fixed_ip_address              = optional(string)
    allowed_address_pairs         = optional(list(object({
      ip_address  = string
      mac_address = string
    })))
  }))
}