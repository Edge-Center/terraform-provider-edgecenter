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
variable "security_groups" {
  description = "A reserved fixed IP is an IP address within a specific network that is reserved for a particular purpose. Reserved fixed IPs are typically not automatically assigned to instances but are instead set aside for specific needs or configurations"
  type = map(object({
    name         = string
    description  = optional(string)
    metadata_map = optional(map(string))
    security_group_rules = optional(list(object({
      direction        = string
      ethertype        = string
      protocol         = string
      port_range_min   = optional(number)
      port_range_max   = optional(number)
      remote_ip_prefix = optional(string)
    })))
  }))
}