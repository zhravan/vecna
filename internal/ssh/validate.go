package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func loadPrivateKey(path string) (ssh.Signer, error) {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, "~") {
		path = filepath.Join(home, strings.TrimLeft(path[1:], `/\\`))
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}

func ValidateConnection(host Host, password string) error {
	var authMethods []ssh.AuthMethod

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if host.IdentityFile != "" {
		if signer, err := loadPrivateKey(host.IdentityFile); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method available (need password or key path)")
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

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            authMethods,
		HostKeyCallback: callback,
		Timeout:         5 * time.Second,
	}
	if len(knownAlgorithms) > 0 {
		config.HostKeyAlgorithms = knownAlgorithms
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("session creation failed: %w", err)
	}
	defer session.Close()

	if err := session.Run("echo 'test'"); err != nil {
		return fmt.Errorf("test command failed: %w", err)
	}

	return nil
}
