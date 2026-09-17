package ssh

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

func loadPrivateKey(path string) (ssh.Signer, error) {
	return loadPrivateKeyFile(path)
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

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            authMethods,
		HostKeyCallback: callback,
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
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

func loadPrivateKeyFile(path string) (ssh.Signer, error) {
	return loadPrivateKeyFromPath(path)
}
