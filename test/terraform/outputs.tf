output "ip" {
  value = libvirt_domain.node.network_interface[0].addresses[0]
}
