package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/retry"
	tgssh "github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"golang.org/x/crypto/ssh"
)

func generateED25519KeyPair(t *testing.T) *tgssh.KeyPair {
	keyPair, err := generateED25519KeyPairE()
	if err != nil {
		t.Fatal(err)
	}
	return keyPair
}

// Terratest contains a utility to generate RSA key pairs. As of OpenSSH 8.8
// ssh-rsa is disabled by default and is considered weak.
// See https://www.openssh.com/txt/release-8.7
// It is inspired by the existing GenerateRSAKeyPair from Terratest.
// See https://github.com/gruntwork-io/terratest/blob/v0.40.12/modules/ssh/key_pair.go
func generateED25519KeyPairE() (*tgssh.KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	keyPKCS8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	keyPEMBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyPKCS8,
	}
	keyPEM := string(pem.EncodeToMemory(keyPEMBlock))

	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	pubKeyString := string(ssh.MarshalAuthorizedKey(sshPubKey))
	return &tgssh.KeyPair{PublicKey: pubKeyString, PrivateKey: keyPEM}, nil
}

func retryApplyUntilIPv4AreAvailable(t *testing.T, tfOptions *terraform.Options, ipsOutputKey string, retries int, sleepBetweenRetries time.Duration) []string {
	result := retry.DoWithRetry(t, "apply until IPv4 is available for all LibVirt domains", retries, sleepBetweenRetries, func() (string, error) {
		terraform.Apply(t, tfOptions)

		sshIPs := terraform.OutputList(t, tfOptions, ipsOutputKey)
		for outputIndex, sshIP := range sshIPs {
			parsedIP := net.ParseIP(sshIP)
			if parsedIP.To4() == nil {
				return "", fmt.Errorf("Output %s[%d]=%s must be an IPv4 address", ipsOutputKey, outputIndex, sshIP)
			}
		}
		return strings.Join(sshIPs, ","), nil
	})
	return strings.Split(result, ",")
}
