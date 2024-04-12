# Kubernetes image builder

## Requirements

### Scope

This repository builds a single machine image, of Rocky 9 with a single Kubernetes version.
Image versions are managed with Git tags.

### Functional requirements

- Machine image must be compatible with KubeVirt.
- Machine image must be compatible with Kubeadm [bootstrap](https://github.com/kubernetes-sigs/cluster-api/tree/main/bootstrap/kubeadm) and [control plane](https://github.com/kubernetes-sigs/cluster-api/tree/main/controlplane/kubeadm) Cluster API providers.
- Machine image must contain QEMU guest agent.
- Machine image must contain Cloud Init.
- Machine image must contain kubeadm, kubelet, kubectl and CNI plugins.
- Machine must be prepared with a firewall setup.

### Nonfunctional requirements

- Automated tests (with Terratest?)
- Support Rocky Linux 9.
- Support only the latest Kubernetes version.

### Testing

- Terratest asserts verifying kubeadm, kubelet, and kubectl versions.
- Terratest test for disk resize.
- Terratest test bootstrapping a cluster with kubeadm and connecting to the Kubernetes API.

## Design

### Requirement prioritization

| Requirement                                   | Priority | Risk |
| --------------------------------------------- | -------- | ---- |
| Compatible with KubeVirt                      | MH       | High |
| Compatible with kubeadm Cluster API providers | MH       | High |
| QEMU guest agent setup                        | MH       | Low  |
| Cloud Init setup                              | MH       | Low  |
| kubeadm installation                          | MH       | High |
| Firewall setup                                | NTH      | Low  |

**Priority key:** MH - must have, NTH - nice to have.

### Functional description

The image builder will be a Packer template.
The Packer template will use [the Ansible provisioner](https://developer.hashicorp.com/packer/integrations/hashicorp/ansible/latest/components/provisioner/ansible) to run an Ansible playbook for configuring the machine.

```mermaid
sequenceDiagram
    # Defined explicitly for nicer order.
    actor Alice
    participant Packer
    participant Ansible

    Alice->>+Packer: packer build
    Packer->>+QEMU/KVM: QEMU builder
    Packer->>+Ansible: Ansible provisioner
    Ansible->>QEMU/KVM: configure VM
    Ansible-->>-Packer: done
    Packer->>QEMU/KVM: export disk image
    QEMU/KVM-->>-Packer: raw image
    Packer-->>-Alice: raw image
```

The image builder will have automated tests written with Terratest in Go.
The test will use Terraform with `dmacvicar/libvirt` provider to provision virtual machines from image artifacts.
Such setup will also allow bootstrapping a cluster over SSH and connecting to the Kubernetes API using the official Go Kubernetes client.

```mermaid
sequenceDiagram
    actor Alice

    Alice->>+Terratest: go test .
    Terratest->>+Packer: packer build
    Packer-->>-Terratest: raw image
    Terratest->>+Terraform: apply test/terraform module
    activate VM
    Terratest-->>VM: assert against SSH command output
    Terratest->>Terraform: destroy test/terrarform module
    deactivate VM
    deactivate Terraform
    Terratest-->>-Alice: results
```

The parts are split into folders in the repository:

- Root folder containing the Packer template.
- `playbook` folder containing an Ansible playbook configuring the image to be a Kubernetes node.
- `test` folder containing Terratest tests.
- `test/terraform` folder containing a Terraform module using `dmacvicar/libvirt` provider for provisioning virtual machines from an image artifact.

## Useful resources

Packer related resources:

- [Rocky Linux downloads.](https://rockylinux.org/download)
- [Packer QEMU builder.](https://developer.hashicorp.com/packer/integrations/hashicorp/qemu/latest/components/builder/qemu)
- [Official image builder project.](https://github.com/kubernetes-sigs/image-builder)
- [Official QEMU image builder.](https://github.com/kubernetes-sigs/image-builder/tree/main/images/capi/packer/qemu)
- [Example Packer repository with automated tests.](https://git.houseofkummer.com/homelab/devops/packer-alpine)
- [Example Libvirt Terraform modules that can be used for testing images.](https://git.houseofkummer.com/Lior/terraform-libvirt-images/-/tree/main?ref_type=heads)
- [KubeVirt Containerized Data Importer lab.](https://kubevirt.io/labs/kubernetes/lab2.html) (How the image artifact will be used)

Kubeadm related resources:

- [Previous POC playbook for preparing Rocky for Kubeadm bootstrap.](https://git.houseofkummer.com/Lior/terraform-libvirt/-/blob/b7241fe100e6f6e5981ce13948d471b83d5325f3/playbook/main.yml)
- [Kubernetes CNI installation from package manager.](https://github.com/kubernetes-sigs/image-builder/blob/main/images/capi/ansible/roles/kubernetes/tasks/redhat.yml#L34) Previous Ansible playbook [downloaded CNI plugins manually.](https://git.houseofkummer.com/Lior/terraform-libvirt/-/blob/b7241fe100e6f6e5981ce13948d471b83d5325f3/playbook/main.yml#L85-102)
- [Kubeadm installation guide.](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)
- [Kubeadm reference documentation.](https://kubernetes.io/docs/reference/setup-tools/kubeadm/)
