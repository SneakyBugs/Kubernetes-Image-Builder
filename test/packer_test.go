package test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/packer"
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
			Args:    []string{"-rf", "../build"},
		})
		packer.BuildArtifact(t, packerOptions)
	}

	sshKeyPair := generateED25519KeyPair(t)

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "terraform",
		Vars: map[string]interface{}{
			"authorized_key": sshKeyPair.PublicKey,
		},
	})

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	sshIP := terraform.Output(t, terraformOptions, "ip")
	host := ssh.Host{
		Hostname:    sshIP,
		SshUserName: "terraform",
		SshKeyPair:  sshKeyPair,
	}

	// Check Cloud Init ran successfully and SSH works.
	ssh.CheckSshCommand(t, host, "cloud-init status --wait")

	containerdShowState := ssh.CheckSshCommand(t, host, "sudo systemctl show containerd --property=ActiveState")
	if !strings.Contains(containerdShowState, "ActiveState=active") {
		t.Fatalf("Expected `systemctl show containerd` output to contain `ActiveState=active`, got: `%s`\n", containerdShowState)
	}

	ssh.CheckSshCommand(t, host, "sudo kubeadm init")
	kubeconfigContent := ssh.FetchContentsOfFile(t, host, true, "/etc/kubernetes/admin.conf")
	kubeconfigFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Fatalf("Expected no error, got %v\n", err)
	}
	_, err = kubeconfigFile.WriteString(kubeconfigContent)
	if err != nil {
		t.Fatalf("Expected no error, got %v\n", err)
	}
	kubeconfig := kubeconfigFile.Name()
	kubeconfigFile.Close()

	kubectlOptions := k8s.NewKubectlOptions("kubernetes-admin@kubernetes", kubeconfig, "kube-system")

	k8s.WaitUntilNumPodsCreated(t, kubectlOptions, v1.ListOptions{}, 7, 10, time.Second*5)

	pods := k8s.ListPods(t, kubectlOptions, v1.ListOptions{})
	for _, pod := range pods {
		if strings.Contains(pod.Name, "coredns") {
			// CoreDNS pods should have "not ready" status because there is no network plugin installed.
			continue
		}
		k8s.WaitUntilPodAvailable(t, kubectlOptions, pod.Name, 10, time.Second*10)
	}
}
