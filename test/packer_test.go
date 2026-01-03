package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPackerImage(t *testing.T) {
	_, ok := os.LookupEnv("TEST_EXISTING_TEMPLATE")
	if !ok {
		packerOptions := &packer.Options{
			Template:   "image.pkr.hcl",
			WorkingDir: "..",
		}

		shell.RunCommand(t, shell.Command{
			Command: "rm",
			Args:    []string{"-rf", "../build", "../packer-manifest.json"},
		})
		packer.BuildArtifact(t, packerOptions)
	}

	sshKeyPair := generateED25519KeyPair(t)
	imagePath := fetchImageNameFromPackerManifest(t)

	authorizedKeys := []string{
		strings.TrimSpace(sshKeyPair.PublicKey),
	}
	additionalKeysValue, ok := os.LookupEnv("TEST_ADDITIONAL_AUTHORIZED_KEYS")
	if ok {
		authorizedKeys = append(authorizedKeys, strings.Split(additionalKeysValue, "\n")...)
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "terraform",
		Vars: map[string]any{
			"image":           imagePath,
			"authorized_keys": authorizedKeys,
		},
	})

	_, ok = os.LookupEnv("TEST_SKIP_DESTROY")
	if !ok {
		defer terraform.Destroy(t, terraformOptions)
	}
	terraform.InitAndApply(t, terraformOptions)

	// Reapply if the output IP is not IPv4 due to a bug with the Libvirt Terraform provider.
	sshIPs := retryApplyUntilIPv4AreAvailable(t, terraformOptions, "ips", 5, time.Second)

	kubeconfigFile, err := os.CreateTemp("", "kubeconfig")
	kubeadmJoinCommand := ""

	for domainIndex, sshIP := range sshIPs {
		host := ssh.Host{
			Hostname:    sshIP,
			SshUserName: "terraform",
			SshKeyPair:  sshKeyPair,
		}

		// Check Cloud Init ran successfully and SSH works.
		// Retry because sometimes SSH server takes a few seconds to start after booting.
		ssh.CheckSshCommandWithRetry(t, host, "cloud-init status --wait", 5, time.Second*5)

		containerdShowState := ssh.CheckSshCommand(t, host, "sudo systemctl show crio.service --property=ActiveState")
		if !strings.Contains(containerdShowState, "ActiveState=active") {
			t.Fatalf("Expected `systemctl show crio.service` output to contain `ActiveState=active`, got: `%s`\n", containerdShowState)
		}

		if domainIndex == 0 {
			// First LibVirt domain is used as control plane node.
			ssh.CheckSshCommand(t, host, "sudo kubeadm init --pod-network-cidr=10.243.0.0/16")

			kubeadmJoinCommand = ssh.CheckSshCommand(t, host, "sudo kubeadm token create --print-join-command")

			kubeconfigContent := ssh.FetchContentsOfFile(t, host, true, "/etc/kubernetes/admin.conf")
			if err != nil {
				t.Fatalf("Expected no error, got %v\n", err)
			}
			_, err = kubeconfigFile.WriteString(kubeconfigContent)
			if err != nil {
				t.Fatalf("Expected no error, got %v\n", err)
			}
			_ = kubeconfigFile.Close()

			// Wait for control plane to be responsive.
			kubectlOptions := k8s.NewKubectlOptions("kubernetes-admin@kubernetes", kubeconfigFile.Name(), "kube-system")
			k8s.WaitUntilPodAvailable(t, kubectlOptions, "kube-apiserver-kib-0", 6, 10*time.Second)
		} else {
			// Remaining LibVirt domains are used as worker nodes.
			o := ssh.CheckSshCommand(t, host, fmt.Sprintf("sudo %s", kubeadmJoinCommand))
			t.Logf(fmt.Sprintf("===== kubeadm join log =====\n%s\n============================\n", o))
		}
	}

	kubectlOptions := k8s.NewKubectlOptions("kubernetes-admin@kubernetes", kubeconfigFile.Name(), "kube-system")

	retry.DoWithRetry(t, "Wait until 2 nodes are connected to the cluster", 18, time.Second*10, func() (string, error) {
		nodes, err := k8s.GetNodesE(t, kubectlOptions)
		if err != nil {
			return "", fmt.Errorf("failed to get nodes: %w", err)
		}
		if len(nodes) != 2 {
			return "", fmt.Errorf("expected 2 nodes, got %d", len(nodes))
		}
		return "", nil
	})

	k8s.WaitUntilNumPodsCreated(t, kubectlOptions, v1.ListOptions{}, 8, 10, time.Second*5)

	kubectlOptions.Namespace = "tigera-operator"
	helm.Install(t, &helm.Options{
		ValuesFiles:    []string{"calico-values.yml"},
		Version:        "v3.30.3",
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--create-namespace", "--repo=https://docs.tigera.io/calico/charts"},
		},
	}, "tigera-operator", "tigera-operator")

	kubectlOptions.Namespace = "cert-manager"
	helm.Install(t, &helm.Options{
		ValuesFiles:    []string{"cert-manager-values.yml"},
		Version:        "v1.19.1",
		KubectlOptions: kubectlOptions,
		ExtraArgs: map[string][]string{
			"install": {"--create-namespace", "--repo=https://charts.jetstack.io"},
		},
	}, "cert-manager", "cert-manager")

	namespaces := k8s.ListNamespaces(t, kubectlOptions, v1.ListOptions{})
	for _, namespace := range namespaces {
		kubectlOptions.Namespace = namespace.Name
		pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{})
		for _, pod := range pods {
			k8s.WaitUntilPodAvailable(t, kubectlOptions, pod.Name, 24, time.Second*10)
		}
	}

	nodes := k8s.GetNodes(t, kubectlOptions)
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	k8s.WaitUntilAllNodesReady(t, kubectlOptions, 5, time.Second*5)
}
