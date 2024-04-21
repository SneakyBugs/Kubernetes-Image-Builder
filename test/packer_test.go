package test

import (
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
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
	// Sometimes the Libvirt provider returns wrong IP address on first apply.
	terraform.Apply(t, terraformOptions)

	sshIP := terraform.Output(t, terraformOptions, "ip")
	host := ssh.Host{
		Hostname:    sshIP,
		SshUserName: "terraform",
		SshKeyPair:  sshKeyPair,
	}

	// Check Cloud Init ran successfully and SSH works.
	ssh.CheckSshCommand(t, host, "cloud-init status --wait")
}
