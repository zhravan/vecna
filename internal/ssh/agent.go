package ssh

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AgentAuth returns an SSH authentication method backed by SSH_AUTH_SOCK.
// The agent is queried lazily, so its socket is not held open for the session.
// Hardware-backed FIDO2/security-key identities work through the agent without
// Vecna handling private key material.
func AgentAuth() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" { return nil, fmt.Errorf("SSH_AUTH_SOCK is not set") }
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		conn, err := net.Dial("unix", sock); if err != nil { return nil, err }; defer conn.Close()
		return agent.NewClient(conn).Signers()
	}), nil
}

func AgentAvailable() bool {
	sock := os.Getenv("SSH_AUTH_SOCK"); if sock == "" { return false }
	conn, err := net.Dial("unix", sock); if err != nil { return false }; defer conn.Close()
	_, err = agent.NewClient(conn).List(); return err == nil
}
