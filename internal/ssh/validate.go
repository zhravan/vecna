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
	expandPath := func(p string) string {
		if strings.HasPrefix(p, "~") {
			return filepath.Join(home, p[1:])
		}
		return p
	}

	keyPath := expandPath(path)
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
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

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
