variable "location" {
  type    = string
  default = "westeurope"
}
variable "prefix" {
  type    = string
  default = "incidentlab"
}
variable "admin_group_object_id" {
  type      = string
  sensitive = true
  default   = ""
}
variable "ssh_public_key" {
  type      = string
  sensitive = true
}
variable "postgres_admin_password" {
  type      = string
  sensitive = true
}
