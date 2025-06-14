variable "node_count" {
  type    = number
  default = 2
}

variable "hostname" {
  type    = string
  default = "kib"
}

variable "user" {
  type    = string
  default = "terraform"
}

variable "authorized_key" {
  type    = string
  default = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICmTzPVmwo0Q7txYnDkD2ubmRxLUBP1MB5x5j8+v0hK8 lior-workstation"
}

variable "memory" {
  type    = number
  default = 1024 * 4
}

variable "cores" {
  type    = number
  default = 4
}

variable "disk_size" {
  type    = number
  default = 1024 * 1024 * 1024 * 20
}

variable "disk_pool" {
  type    = string
  default = "default"
}
