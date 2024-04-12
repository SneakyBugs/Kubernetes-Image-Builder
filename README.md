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

## Useful resources

Packer related resources:

- [Rocky Linux downloads.](https://rockylinux.org/download)
- [Packer QEMU builder.](https://developer.hashicorp.com/packer/integrations/hashicorp/qemu/latest/components/builder/qemu)
- [Official image builder project.](https://github.com/kubernetes-sigs/image-builder)
- [Official QEMU image builder.](https://github.com/kubernetes-sigs/image-builder/tree/main/images/capi/packer/qemu)
- [Example Packer repository with automated tests.](https://git.houseofkummer.com/homelab/devops/packer-alpine)

Kubeadm related resources:

- [Previous POC playbook for preparing Rocky for Kubeadm bootstrap.](https://git.houseofkummer.com/Lior/terraform-libvirt/-/blob/b7241fe100e6f6e5981ce13948d471b83d5325f3/playbook/main.yml)
- [Kubernetes CNI installation from package manager.](https://github.com/kubernetes-sigs/image-builder/blob/main/images/capi/ansible/roles/kubernetes/tasks/redhat.yml#L34) Previous Ansible playbook [downloaded CNI plugins manually.](https://git.houseofkummer.com/Lior/terraform-libvirt/-/blob/b7241fe100e6f6e5981ce13948d471b83d5325f3/playbook/main.yml#L85-102)
- [Kubeadm installation guide.](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)
- [Kubeadm reference documentation.](https://kubernetes.io/docs/reference/setup-tools/kubeadm/)
