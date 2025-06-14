output "ips" {
  value = [for domain in libvirt_domain.node : domain.network_interface[0].addresses[0]]
}
