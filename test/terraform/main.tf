locals {
  nodes = toset([for num in range(var.node_count) : tostring(num)])
}

resource "libvirt_pool" "local" {
  name = "image_builder"
  type = "dir"
  target {
    path = "./pool"
  }
}

resource "libvirt_network" "local" {
  name = "image-builder"
  addresses = [
    "172.16.0.0/24"
  ]
}

resource "random_uuid" "domain" {}

resource "libvirt_volume" "disk" {
  for_each = local.nodes
  name     = "${var.hostname}-${random_uuid.domain.result}-${each.key}.qcow2"
  pool     = libvirt_pool.local.name
  source   = "../../build/rocky-9.qcow2"
}

resource "libvirt_cloudinit_disk" "config" {
  for_each  = local.nodes
  name      = "${var.hostname}-${random_uuid.domain.result}-${each.key}-cloudinit.iso"
  pool      = libvirt_pool.local.name
  user_data = <<-EOF
    #cloud-config
    users:
      - name: ${var.user}
        sudo: ALL=(ALL) NOPASSWD:ALL
        shell: /bin/bash
        ssh_authorized_keys:
          - ${var.authorized_key}
          - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICmTzPVmwo0Q7txYnDkD2ubmRxLUBP1MB5x5j8+v0hK8 lior-workstation
    hostname: ${var.hostname}-${each.key}
    prefer_fqdn_over_hostname: false
    write_files:
      - path: /etc/containers/registries.conf
        content: |
          [[registry]]
          prefix = "docker.io"
          location = "oci.infra.sneakybugs.com/docker"
    EOF
}

resource "libvirt_domain" "node" {
  for_each   = local.nodes
  name       = "${var.hostname}-${random_uuid.domain.result}-${each.key}"
  memory     = var.memory
  vcpu       = var.cores
  cloudinit  = libvirt_cloudinit_disk.config[each.key].id
  qemu_agent = true
  firmware   = "/usr/share/OVMF/OVMF_CODE.fd"

  # Required for idempotency.
  nvram {
    file     = "/var/lib/libvirt/qemu/nvram/${var.hostname}-${random_uuid.domain.result}-${each.key}_VARS.fd"
    template = "/usr/share/OVMF/OVMF_VARS.fd"
  }

  # Rocky requires CPU features not supported by default qemu64 mode.
  cpu {
    mode = "host-passthrough"
  }

  network_interface {
    network_name   = libvirt_network.local.name
    wait_for_lease = true
  }

  disk {
    volume_id = libvirt_volume.disk[each.key].id
  }

  console {
    type        = "pty"
    target_port = "0"
  }

  graphics {
    type        = "vnc"
    listen_type = "address"
    autoport    = true
  }

  # Modify Cloud Init cdrom drive to use sata instead of ide.
  # Cloud init in Rocky 9 failed to find the drive with ide bus type.
  xml {
    xslt = <<-EOF
      <xsl:stylesheet version="1.0" 
       xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
          <xsl:output omit-xml-declaration="yes" indent="yes"/>

          <xsl:param name="pNewType" select="'sata'"/>

          <xsl:template match="node()|@*">
              <xsl:copy>
                  <xsl:apply-templates select="node()|@*"/>
              </xsl:copy>
          </xsl:template>

          <xsl:template match="disk[@device='cdrom']/target/@bus">
              <xsl:attribute name="bus">
                  <xsl:value-of select="$pNewType"/>
              </xsl:attribute>
          </xsl:template>
      </xsl:stylesheet>
      EOF
  }
}
