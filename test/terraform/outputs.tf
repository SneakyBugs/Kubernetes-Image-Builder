output "ips" {
  value = [for domain in libvirt_domain.node : try(domain.network_interface[0].addresses[0], null)]
}
