variable "project_id" {
  description = "The ID of the project."
  default = ""
}
variable "keypair_name" {
  description = "The name of the key pair to be associated with the instance for SSH access."
  default     = ""
}
variable "permanent_api_token" {
  description = "A permanent API-token."
  default     = ""
}