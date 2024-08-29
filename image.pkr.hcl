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

source "qemu" "rocky" {
  iso_url          = "https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base-9.4-20240509.0.x86_64.qcow2"
  iso_checksum     = "file:https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base-9.4-20240509.0.x86_64.qcow2.CHECKSUM"
  disk_image       = true
  skip_resize_disk = true
  headless         = true
  vm_name          = "rocky-9.qcow2"

  efi_boot  = true
  cpu_model = "host"

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
    # https://github.com/hashicorp/packer/issues/11783
    extra_arguments = ["--scp-extra-args", "'-O'"]
  }

  provisioner "shell" {
    inline = [
      # Cleanup for systemd. See:
      # https://systemd.io/BUILDING_IMAGES/
      "sudo rm /var/lib/systemd/random-seed /etc/hostname",
      "sudo cloud-init clean --logs --seed --machine-id",
    ]
  }
}
