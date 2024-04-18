terraform {
  required_version = ">=1.6"
  required_providers {
    libvirt = {
      source  = "dmacvicar/libvirt"
      version = "0.7.6"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.6.0"
    }
  }
}

provider "libvirt" {
  uri = "qemu:///system"
}
