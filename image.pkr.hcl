packer {
  required_plugins {
    qemu = {
      source  = "github.com/hashicorp/qemu"
      version = "~> 1"
    }
    ansible = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

locals {
  rocky_version      = "10.1"
  rocky_build        = "20251116.0"
  kubernetes_version = "1.35.0"
  cni_version        = "1.8.0"
  rocky_major        = split(".", local.rocky_version)[0]
}

source "qemu" "rocky" {
  iso_url          = "https://dl.rockylinux.org/pub/rocky/${local.rocky_version}/images/x86_64/Rocky-${local.rocky_major}-GenericCloud-Base-${local.rocky_version}-${local.rocky_build}.x86_64.qcow2"
  iso_checksum     = "file:https://dl.rockylinux.org/pub/rocky/${local.rocky_version}/images/x86_64/Rocky-${local.rocky_major}-GenericCloud-Base-${local.rocky_version}-${local.rocky_build}.x86_64.qcow2.CHECKSUM"
  disk_image       = true
  skip_resize_disk = true
  headless         = true
  vm_name          = "rocky-${local.rocky_version}-k8s-${local.kubernetes_version}-${formatdate("YYYYMMDD", timestamp())}.qcow2"

  efi_firmware_code = "/usr/share/OVMF/OVMF_CODE_4M.fd"
  efi_firmware_vars = "/usr/share/OVMF/OVMF_VARS_4M.fd"
  cpu_model         = "host"

  cpus   = 4
  memory = 2048

  output_directory = "build"
  format           = "qcow2"

  boot_wait        = "30s"
  ssh_username     = "packer"
  ssh_password     = "packer"
  shutdown_command = "echo 'packer' | sudo -S shutdown -P now"

  # Cloud Init NoCloud drive.
  # https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html
  cd_label = "cidata"
  cd_content = {
    "meta-data" = ""
    "user-data" = <<-EOF
      #cloud-config
      users:
      - name: packer
        sudo: ALL=(ALL) NOPASSWD:ALL
        shell: /bin/bash
        lock_passwd: false
        plain_text_passwd: packer
      ssh_pwauth: true
      hostname: template
    EOF
  }
}

build {
  sources = ["source.qemu.rocky"]

  provisioner "ansible" {
    command       = "./ansible/ansible.sh"
    playbook_file = "./ansible/main.yml"
    user          = "packer"
    extra_arguments = [
      # https://github.com/hashicorp/packer/issues/11783
      "--scp-extra-args", "'-O'",
      "--extra-vars", "template_kubernetes_version=${local.kubernetes_version} template_cni_version=${local.cni_version}"
    ]
  }

  provisioner "shell" {
    inline = [
      # Cleanup for systemd. See:
      # https://systemd.io/BUILDING_IMAGES/
      "sudo rm /var/lib/systemd/random-seed /etc/hostname",
      "sudo cloud-init clean --logs --seed --machine-id",
    ]
  }

  post-processor "manifest" {}
}
