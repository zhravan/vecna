package ssh

import (
	"fmt"
	"net"
	"os"
	"runtime"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AgentAuth returns an SSH authentication method backed by the user's
// SSH_AUTH_SOCK. Hardware-backed FIDO2/security-key identities work through
// the agent without Vecna ever handling the private key material.
func AgentAuth() (ssh.AuthMethod, func() error, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, fmt.Errorf("SSH_AUTH_SOCK is not set")
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to ssh-agent: %w", err)
	}
	ag := agent.NewClient(conn)
	closeFn := func() error { return conn.Close() }
	return ssh.PublicKeysCallback(ag.Signers), closeFn, nil
}

// AgentAvailable reports whether the configured SSH agent can be reached.
func AgentAvailable() bool {
	if os.Getenv("SSH_AUTH_SOCK") == "" || runtime.GOOS == "windows" {
		return false
	}
	auth, closeFn, err := AgentAuth()
	if err != nil || auth == nil {
		return false
	}
	_ = closeFn()
	return true
}
