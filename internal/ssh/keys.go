package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

func GenerateKeyPair(name string) (privatePath, publicPath string, err error) {
	home, _ := os.UserHomeDir()
	keyDir := filepath.Join(home, ".ssh", "vecna")
	os.MkdirAll(keyDir, 0700)

	privatePath = filepath.Join(keyDir, name)
	publicPath = privatePath + ".pub"

	if _, err := os.Stat(privatePath); err == nil {
		return privatePath, publicPath, nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	privateFile, err := os.OpenFile(privatePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("failed to create private key: %w", err)
	}
	defer privateFile.Close()
	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create public key: %w", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	if err := os.WriteFile(publicPath, publicKeyBytes, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write public key: %w", err)
	}
	return privatePath, publicPath, nil
}

func DeployPublicKey(host Host, password, publicKeyPath string) error {
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	callback, err := HostKeyCallback()
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	knownAlgorithms, err := HostKeyAlgorithms(addr)
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{User: host.User, Auth: []ssh.AuthMethod{ssh.Password(password)}, HostKeyCallback: callback}
	if len(knownAlgorithms) > 0 {
		config.HostKeyAlgorithms = knownAlgorithms
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	publicKeyStr := strings.TrimSpace(string(publicKey))
	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr
	cmd := fmt.Sprintf(`
		mkdir -p ~/.ssh &&
		chmod 700 ~/.ssh &&
		if [ ! -f ~/.ssh/authorized_keys ]; then
			touch ~/.ssh/authorized_keys &&
			chmod 600 ~/.ssh/authorized_keys;
		fi &&
		if ! grep -qF "%s" ~/.ssh/authorized_keys; then
			echo "%s" >> ~/.ssh/authorized_keys &&
			chmod 600 ~/.ssh/authorized_keys;
		fi
	`, publicKeyStr, publicKeyStr)

	if err := session.Run(cmd); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return fmt.Errorf("failed to deploy key: %w (output: %s)", err, errMsg)
	}

	verifySession, err := client.NewSession()
	if err == nil {
		defer verifySession.Close()
		var verifyStdout, verifyStderr strings.Builder
		verifySession.Stdout = &verifyStdout
		verifySession.Stderr = &verifyStderr
		verifyCmd := fmt.Sprintf(`grep -qF "%s" ~/.ssh/authorized_keys`, publicKeyStr)
		if err := verifySession.Run(verifyCmd); err != nil {
			return fmt.Errorf("key deployment verification failed: key not found in authorized_keys")
		}
	}
	return nil
}
