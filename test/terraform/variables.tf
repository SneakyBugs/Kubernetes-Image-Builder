variable "image" {
  type = string
}

variable "authorized_keys" {
  type = list(string)
}

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
